package cluster

import (
	"time"

	"github.com/danielfrg/homelab-console/internal/apps"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var testNow = time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)

func ptr[T any](v T) *T { return &v }

type resourceRef struct{ kind, namespace, name string }

func application(name, sync, health string, annotations map[string]string, resources ...resourceRef) unstructured.Unstructured {
	items := []any{}
	for _, r := range resources {
		group := "apps"
		if r.kind == "HTTPRoute" {
			group = "gateway.networking.k8s.io"
		}
		items = append(items, map[string]any{"group": group, "version": "v1", "kind": r.kind, "namespace": r.namespace, "name": r.name})
	}
	obj := unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "argoproj.io/v1alpha1",
		"kind":       "Application",
		"metadata":   map[string]any{"name": name, "namespace": "argocd"},
		"spec": map[string]any{
			"project":     "default",
			"destination": map[string]any{"namespace": "default"},
			"source":      map[string]any{"repoURL": "git@github.com:danielfrg/homelab.git", "path": "manifests/" + name},
		},
		"status": map[string]any{
			"sync":      map[string]any{"status": sync, "revision": "abc1234def"},
			"health":    map[string]any{"status": health},
			"resources": items,
			"history":   []any{map[string]any{"revision": "abc1234def", "deployedAt": "2026-10-21T02:32:31Z"}},
			"summary":   map[string]any{"images": []any{"ghcr.io/example/" + name + ":1"}},
			"operationState": map[string]any{
				"phase": "Succeeded", "message": "successfully synced", "finishedAt": "2026-10-21T02:32:31Z",
			},
		},
	}}
	if annotations != nil {
		obj.SetAnnotations(annotations)
	}
	return obj
}

func deployment(namespace, name string, desired, ready, updated int32) appsv1.Deployment {
	return appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name, Generation: 2},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr(desired),
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/name": name}},
			Strategy: appsv1.DeploymentStrategy{Type: appsv1.RollingUpdateDeploymentStrategyType},
		},
		Status: appsv1.DeploymentStatus{ObservedGeneration: 2, Replicas: desired, ReadyReplicas: ready, UpdatedReplicas: updated},
	}
}

func pod(namespace, name, appLabel string, started time.Time) corev1.Pod {
	return corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name, Labels: map[string]string{"app.kubernetes.io/name": appLabel}, CreationTimestamp: metav1.NewTime(started)},
		Spec:       corev1.PodSpec{NodeName: "srv-nuc-02", Containers: []corev1.Container{{Name: appLabel, Image: "ghcr.io/example/" + appLabel + ":1"}}},
		Status: corev1.PodStatus{
			Phase:             corev1.PodRunning,
			StartTime:         ptr(metav1.NewTime(started)),
			Conditions:        []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionTrue, LastTransitionTime: metav1.NewTime(started)}},
			ContainerStatuses: []corev1.ContainerStatus{{Name: appLabel, Ready: true, Image: "ghcr.io/example/" + appLabel + ":1"}},
		},
	}
}

func crashing(p corev1.Pod) corev1.Pod {
	p.Status.Conditions[0].Status = corev1.ConditionFalse
	p.Status.ContainerStatuses[0].Ready = false
	p.Status.ContainerStatuses[0].RestartCount = 7
	p.Status.ContainerStatuses[0].State = corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"}}
	p.Status.ContainerStatuses[0].LastTerminationState = corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Reason: "Error", ExitCode: 1, FinishedAt: metav1.NewTime(testNow.Add(-time.Minute))}}
	return p
}

func node(name string, ready bool) corev1.Node {
	status := corev1.ConditionTrue
	if !ready {
		status = corev1.ConditionFalse
	}
	return corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: name, Labels: map[string]string{"homelab.danielfrg.net/role": "compute"}},
		Status: corev1.NodeStatus{
			Allocatable: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("4"), corev1.ResourceMemory: resource.MustParse("16Gi")},
			Conditions:  []corev1.NodeCondition{{Type: corev1.NodeReady, Status: status}},
			NodeInfo:    corev1.NodeSystemInfo{KubeletVersion: "v1.34.3+k3s1", KernelVersion: "6.18.2-arch2-1"},
		},
	}
}

// fixture mirrors the shape of the real cluster: a multi-workload app, an opted-out
// OnDelete StatefulSet, a crashing single-workload app, and a pod outside any app.
func fixture() Raw {
	optOut := map[string]string{"homelab-console/restart": "false"}
	vault := appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Namespace: "vault", Name: "vault", Generation: 1},
		Spec: appsv1.StatefulSetSpec{
			Replicas:       ptr(int32(1)),
			Selector:       &metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/name": "vault"}},
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{Type: appsv1.OnDeleteStatefulSetStrategyType},
		},
		Status: appsv1.StatefulSetStatus{ObservedGeneration: 1, Replicas: 1, ReadyReplicas: 1, UpdatedReplicas: 1},
	}
	plex := deployment("default", "plex", 1, 1, 1)
	plex.Annotations = optOut
	started := testNow.Add(-48 * time.Hour)
	return Raw{
		Applications: []unstructured.Unstructured{
			application("home-assistant", "Synced", "Healthy", nil,
				resourceRef{"Deployment", "default", "frigate"}, resourceRef{"Deployment", "default", "home-assistant"},
				resourceRef{"HTTPRoute", "default", "home-assistant-http"}),
			application("vault", "Synced", "Healthy", optOut, resourceRef{"StatefulSet", "vault", "vault"}),
			application("sofigram", "Synced", "Degraded", nil, resourceRef{"Deployment", "default", "sofigram"}),
			application("plex", "OutOfSync", "Healthy", nil, resourceRef{"Deployment", "default", "plex"}),
		},
		Deployments: []appsv1.Deployment{
			deployment("default", "frigate", 1, 1, 1),
			deployment("default", "home-assistant", 1, 1, 1),
			deployment("default", "sofigram", 1, 0, 1),
			plex,
			deployment("kube-system", "coredns", 1, 1, 1),
		},
		StatefulSets: []appsv1.StatefulSet{vault},
		Pods: []corev1.Pod{
			pod("default", "frigate-abc", "frigate", started),
			pod("default", "home-assistant-abc", "home-assistant", started),
			crashing(pod("default", "sofigram-abc", "sofigram", started)),
			pod("default", "plex-abc", "plex", started),
			pod("vault", "vault-0", "vault", started),
			pod("kube-system", "coredns-abc", "coredns", started),
		},
		Nodes:     []corev1.Node{node("srv-nuc-01", true), node("srv-nuc-02", true)},
		NodeUsage: map[string]NodeUsage{"srv-nuc-01": {CPUMilli: 500, MemoryBytes: 4 << 30}, "srv-nuc-02": {CPUMilli: 1000, MemoryBytes: 8 << 30}},
		Events: []corev1.Event{{
			ObjectMeta:     metav1.ObjectMeta{Namespace: "default", Name: "sofigram-abc.1"},
			InvolvedObject: corev1.ObjectReference{Kind: "Pod", Namespace: "default", Name: "sofigram-abc"},
			Type:           corev1.EventTypeWarning, Reason: "BackOff", Message: "Back-off restarting failed container", Count: 12,
			LastTimestamp: metav1.NewTime(testNow.Add(-time.Minute)),
		}},
		Links: []apps.App{{ID: "default/home-assistant-http", Name: "Home Assistant", URL: "https://ha.corp.danielfrg.net", Aliases: []string{"home.corp.danielfrg.net"}, Group: "Home", Icon: "home-assistant", Description: "Smart home"}},
	}
}

func schemaGR() schema.GroupResource {
	return schema.GroupResource{Group: "apps", Resource: "deployments"}
}
