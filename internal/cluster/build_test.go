package cluster

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func findApp(t *testing.T, s Snapshot, name string) App {
	t.Helper()
	for _, a := range s.Apps {
		if a.Name == name {
			return a
		}
	}
	t.Fatalf("app %q missing", name)
	return App{}
}

func findWorkload(t *testing.T, s Snapshot, namespace, name string) Workload {
	t.Helper()
	for _, w := range s.Workloads {
		if w.Namespace == namespace && w.Name == name {
			return w
		}
	}
	t.Fatalf("workload %s/%s missing", namespace, name)
	return Workload{}
}

func findPod(t *testing.T, s Snapshot, namespace, name string) Pod {
	t.Helper()
	for _, p := range s.Pods {
		if p.Namespace == namespace && p.Name == name {
			return p
		}
	}
	t.Fatalf("pod %s/%s missing", namespace, name)
	return Pod{}
}

func TestBuildJoinsAppsWorkloadsPodsLinks(t *testing.T) {
	s := Build(fixture(), testNow)
	ha := findApp(t, s, "home-assistant")
	if !reflect.DeepEqual(ha.Workloads, []string{"default/frigate", "default/home-assistant"}) {
		t.Fatalf("workloads %v", ha.Workloads)
	}
	if len(ha.Links) != 1 || ha.Links[0].URL != "https://ha.corp.danielfrg.net" || ha.Links[0].Aliases[0] != "home.corp.danielfrg.net" {
		t.Fatalf("links %+v", ha.Links)
	}
	if ha.Sync != "Synced" || ha.Health != "Healthy" || ha.Revision != "abc1234def" || ha.DeployedAt == nil || ha.Project != "default" {
		t.Fatalf("app %+v", ha)
	}
	frigate := findPod(t, s, "default", "frigate-abc")
	if frigate.App != "home-assistant" || frigate.Workload != "frigate" || !frigate.Ready || frigate.Node != "srv-nuc-02" {
		t.Fatalf("pod %+v", frigate)
	}
	w := findWorkload(t, s, "default", "frigate")
	if w.App != "home-assistant" || !reflect.DeepEqual(w.Pods, []string{"frigate-abc"}) || w.Desired != 1 || w.Ready != 1 {
		t.Fatalf("workload %+v", w)
	}
	if !s.ArgoAvailable || !s.MetricsAvailable || !s.UpdatedAt.Equal(testNow) {
		t.Fatalf("flags %+v", s)
	}
}

func TestBuildRestartability(t *testing.T) {
	s := Build(fixture(), testNow)
	if w := findWorkload(t, s, "default", "frigate"); !w.Restartable || w.NotRestartableReason != "" {
		t.Fatalf("frigate %+v", w)
	}
	vault := findWorkload(t, s, "vault", "vault")
	if vault.Restartable || !(strings.Contains(vault.NotRestartableReason, "OnDelete") || strings.Contains(vault.NotRestartableReason, "opted out")) {
		t.Fatalf("vault %+v", vault)
	}
	if plex := findWorkload(t, s, "default", "plex"); plex.Restartable || !strings.Contains(plex.NotRestartableReason, "opted out") {
		t.Fatalf("plex %+v", plex)
	}
	if coredns := findWorkload(t, s, "kube-system", "coredns"); coredns.Restartable || coredns.App != "" {
		t.Fatalf("coredns %+v", coredns)
	}
	for _, w := range s.Workloads {
		if w.Kind == "DaemonSet" {
			t.Fatal("DaemonSets must not appear")
		}
	}
}

func TestBuildPodProblems(t *testing.T) {
	raw := fixture()
	pending := pod("default", "pending-long", "nothing", testNow.Add(-6*time.Minute))
	pending.Status.Phase = corev1.PodPending
	pending.Status.Conditions = nil
	pending.Status.ContainerStatuses = nil
	fresh := pending
	fresh.Name = "pending-fresh"
	fresh.CreationTimestamp = metav1.NewTime(testNow.Add(-time.Minute))
	recentOOM := pod("default", "oom-recent", "nothing", testNow.Add(-48*time.Hour))
	recentOOM.Status.ContainerStatuses[0].LastTerminationState = corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Reason: "OOMKilled", ExitCode: 137, FinishedAt: metav1.NewTime(testNow.Add(-2 * time.Hour))}}
	oldOOM := recentOOM
	oldOOM.Name = "oom-old"
	oldOOM.Status.ContainerStatuses = []corev1.ContainerStatus{recentOOM.Status.ContainerStatuses[0]}
	oldOOM.Status.ContainerStatuses[0].LastTerminationState = corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Reason: "OOMKilled", ExitCode: 137, FinishedAt: metav1.NewTime(testNow.Add(-30 * time.Hour))}}
	raw.Pods = append(raw.Pods, pending, fresh, recentOOM, oldOOM)
	s := Build(raw, testNow)
	cases := map[string]string{"sofigram-abc": "CrashLoopBackOff", "pending-long": "Pending", "pending-fresh": "", "oom-recent": "OOMKilled", "oom-old": "", "frigate-abc": ""}
	for name, want := range cases {
		if got := findPod(t, s, "default", name).Problem; got != want {
			t.Errorf("%s problem %q want %q", name, got, want)
		}
	}
	crash := findPod(t, s, "default", "sofigram-abc")
	if crash.Restarts != 7 || crash.Waiting != "CrashLoopBackOff" || crash.LastTermination == nil || crash.LastTermination.ExitCode != 1 {
		t.Fatalf("crash %+v", crash)
	}
}

func TestBuildRollout(t *testing.T) {
	raw := fixture()
	stuck := deployment("default", "frigate", 1, 0, 1)
	stuck.Status.Conditions = []appsv1.DeploymentCondition{{Type: "Progressing", Status: corev1.ConditionFalse, Reason: "ProgressDeadlineExceeded"}}
	raw.Deployments[0] = stuck
	raw.Deployments[1] = deployment("default", "home-assistant", 2, 1, 1)
	s := Build(raw, testNow)
	if got := findWorkload(t, s, "default", "frigate").Rollout; got != "stuck" {
		t.Fatalf("frigate rollout %q", got)
	}
	if got := findWorkload(t, s, "default", "home-assistant").Rollout; got != "progressing" {
		t.Fatalf("home-assistant rollout %q", got)
	}
	if got := findWorkload(t, s, "default", "sofigram").Rollout; got != "complete" {
		t.Fatalf("sofigram rollout %q", got)
	}
}

func TestBuildNodes(t *testing.T) {
	s := Build(fixture(), testNow)
	if len(s.Nodes) != 2 || s.Nodes[0].CPUPercent == nil || *s.Nodes[0].CPUPercent != 12.5 || *s.Nodes[0].MemoryPercent != 25 {
		t.Fatalf("nodes %+v", s.Nodes)
	}
	if s.Nodes[0].Role != "compute" || s.Nodes[0].KernelVersion != "6.18.2-arch2-1" || !s.Nodes[0].Ready {
		t.Fatalf("node %+v", s.Nodes[0])
	}
	raw := fixture()
	raw.NodeUsage = nil
	raw.MetricsErr = errors.New("no metrics")
	s = Build(raw, testNow)
	if s.MetricsAvailable || s.Nodes[0].CPUPercent != nil || s.Nodes[0].MemoryPercent != nil {
		t.Fatalf("nodes without metrics %+v", s.Nodes[0])
	}
}

func TestBuildOrphanPodHasNoApp(t *testing.T) {
	s := Build(fixture(), testNow)
	if p := findPod(t, s, "kube-system", "coredns-abc"); p.App != "" || p.Workload != "coredns" {
		t.Fatalf("coredns pod %+v", p)
	}
}
