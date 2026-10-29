package cluster

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

type funcReader struct {
	raw   func() Raw
	reads int
}

func (r *funcReader) Read(context.Context) (Raw, error) { r.reads++; return r.raw(), nil }

type consoleRig struct {
	console *Console
	typed   *fake.Clientset
	reader  *funcReader
	now     *time.Time
	logOpts *corev1.PodLogOptions
	logPod  string
	logBody string
}

func newRig(t *testing.T, enabled bool, mutate func(*Raw)) *consoleRig {
	t.Helper()
	rig := &consoleRig{typed: fakeTyped(), logBody: "line1\nline2\n"}
	now := testNow
	rig.now = &now
	rig.reader = &funcReader{raw: func() Raw {
		raw := fixture()
		if mutate != nil {
			mutate(&raw)
		}
		return raw
	}}
	inv := NewInventory(rig.reader, nil)
	inv.now = func() time.Time { return *rig.now }
	rig.console = NewConsole(inv, rig.typed, enabled, func() time.Time { return *rig.now })
	rig.console.logStream = func(_ context.Context, namespace, pod string, opts *corev1.PodLogOptions) (io.ReadCloser, error) {
		rig.logPod, rig.logOpts = namespace+"/"+pod, opts
		return io.NopCloser(strings.NewReader(rig.logBody)), nil
	}
	return rig
}

func userMessage(t *testing.T, err error) string {
	t.Helper()
	var ue *UserError
	if !errors.As(err, &ue) {
		t.Fatalf("want UserError, got %v", err)
	}
	return ue.Message
}

func restartedAt(t *testing.T, rig *consoleRig, namespace, name string) string {
	t.Helper()
	d, err := rig.typed.AppsV1().Deployments(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	return d.Spec.Template.Annotations[restartAnnotation]
}

func TestLogsDefaultsAndBounds(t *testing.T) {
	second := pod("default", "frigate-new", "frigate", testNow.Add(-time.Hour))
	rig := newRig(t, true, func(raw *Raw) { raw.Pods = append(raw.Pods, second) })
	ctx := context.Background()
	out, err := rig.console.Logs(ctx, LogsRequest{Workload: "frigate"})
	if err != nil || *rig.logOpts.TailLines != 100 || rig.logPod != "default/frigate-new" || rig.logOpts.Container != "frigate" {
		t.Fatal(out, err, rig.logPod)
	}
	if out.Pod != "frigate-new" || len(out.OtherPods) != 1 || out.OtherPods[0] != "frigate-abc" || out.Text != "line1\nline2\n" || out.Truncated {
		t.Fatalf("%+v", out)
	}
	if _, err = rig.console.Logs(ctx, LogsRequest{Pod: "frigate-abc", Lines: 501}); userMessage(t, err) == "" {
		t.Fatal("expected bound error")
	}
	rig.logBody = strings.Repeat("x", 70000)
	out, err = rig.console.Logs(ctx, LogsRequest{Pod: "default/frigate-abc", Lines: 500})
	if err != nil || len(out.Text) != 65536 || !out.Truncated || *rig.logOpts.TailLines != 500 {
		t.Fatal(len(out.Text), out.Truncated, err)
	}
	if _, err = rig.console.Logs(ctx, LogsRequest{}); userMessage(t, err) == "" {
		t.Fatal("expected missing target error")
	}
	if _, err = rig.console.Logs(ctx, LogsRequest{Pod: "nope"}); userMessage(t, err) == "" {
		t.Fatal("expected not found")
	}
}

func TestLogsPreviousUnavailable(t *testing.T) {
	rig := newRig(t, true, nil)
	_, err := rig.console.Logs(context.Background(), LogsRequest{Workload: "frigate", Previous: true})
	if !strings.Contains(userMessage(t, err), "no previous container logs") {
		t.Fatal(err)
	}
	if _, err = rig.console.Logs(context.Background(), LogsRequest{Workload: "sofigram", Previous: true}); err != nil || !rig.logOpts.Previous {
		t.Fatal(err)
	}
}

func TestRestartWorkload(t *testing.T) {
	rig := newRig(t, true, nil)
	out, err := rig.console.Restart(context.Background(), RestartRequest{Workload: "frigate", Reason: "cameras frozen", Source: "mcp"})
	if err != nil || len(out.Restarted) != 1 || out.Restarted[0].Name != "frigate" || out.Message != "Rollout started; check get_app for progress." {
		t.Fatal(out, err)
	}
	if got := restartedAt(t, rig, "default", "frigate"); got != testNow.Format(time.RFC3339) {
		t.Fatalf("annotation %q", got)
	}
	events, _ := rig.typed.CoreV1().Events("default").List(context.Background(), metav1.ListOptions{})
	found := false
	for _, e := range events.Items {
		if e.Reason == "Restarted" && e.InvolvedObject.Name == "frigate" && strings.Contains(e.Message, "mcp") && strings.Contains(e.Message, "cameras frozen") {
			found = true
		}
	}
	if !found {
		t.Fatalf("no restart event in %+v", events.Items)
	}
	reads := rig.reader.reads
	if _, err = rig.console.Snapshot(context.Background(), false); err != nil || rig.reader.reads != reads+1 {
		t.Fatal("restart should invalidate the snapshot")
	}
}

func TestRestartWorkloadNamedLikeApp(t *testing.T) {
	rig := newRig(t, true, nil)
	if _, err := rig.console.Restart(context.Background(), RestartRequest{Workload: "home-assistant", Source: "ui"}); err != nil {
		t.Fatal(err)
	}
	if restartedAt(t, rig, "default", "home-assistant") == "" || restartedAt(t, rig, "default", "frigate") != "" {
		t.Fatal("only the home-assistant Deployment may restart")
	}
}

func TestRestartApp(t *testing.T) {
	rig := newRig(t, true, nil)
	out, err := rig.console.Restart(context.Background(), RestartRequest{App: "home-assistant", Source: "mcp"})
	if err != nil || len(out.Restarted) != 2 {
		t.Fatal(out, err)
	}
	if restartedAt(t, rig, "default", "frigate") == "" || restartedAt(t, rig, "default", "home-assistant") == "" {
		t.Fatal("both workloads should restart")
	}
	_, err = rig.console.Restart(context.Background(), RestartRequest{App: "frigate", Source: "mcp"})
	if msg := userMessage(t, err); msg != "frigate is a workload in app home-assistant; pass workload" {
		t.Fatal(msg)
	}
}

func TestRestartRefusals(t *testing.T) {
	rig := newRig(t, true, func(raw *Raw) {
		raw.Applications = append(raw.Applications, application("orphan-app", "Synced", "Healthy", nil))
	})
	ctx := context.Background()
	for _, req := range []RestartRequest{
		{Workload: "vault"}, {Workload: "coredns"}, {Workload: "plex"}, {App: "vault"},
		{}, {Workload: "frigate", App: "home-assistant"}, {Workload: "nope"}, {App: "nope"},
	} {
		if _, err := rig.console.Restart(ctx, req); userMessage(t, err) == "" {
			t.Fatalf("%+v should be refused", req)
		}
	}
	if msg := userMessage(t, func() error { _, err := rig.console.Restart(ctx, RestartRequest{Workload: "vault"}); return err }()); !strings.Contains(msg, "OnDelete") && !strings.Contains(msg, "opted out") {
		t.Fatal(msg)
	}
	off := newRig(t, false, nil)
	if _, err := off.console.Restart(ctx, RestartRequest{Workload: "frigate"}); userMessage(t, err) != "restart is disabled" {
		t.Fatal(err)
	}
	if off.console.RestartEnabled() || !rig.console.RestartEnabled() {
		t.Fatal("RestartEnabled")
	}
}

func TestRestartAllowsCrashingWorkload(t *testing.T) {
	rig := newRig(t, true, nil)
	if _, err := rig.console.Restart(context.Background(), RestartRequest{Workload: "sofigram"}); err != nil {
		t.Fatal(err)
	}
}

func TestRestartCooldown(t *testing.T) {
	rolling := newRig(t, true, func(raw *Raw) { raw.Deployments[0].Status.UpdatedReplicas = 0 })
	_, err := rolling.console.Restart(context.Background(), RestartRequest{Workload: "frigate"})
	if !strings.Contains(userMessage(t, err), "rollout in progress") {
		t.Fatal(err)
	}
	rig := newRig(t, true, nil)
	for i := 0; i < 3; i++ {
		if _, err := rig.console.Restart(context.Background(), RestartRequest{Workload: "frigate"}); err != nil {
			t.Fatal(i, err)
		}
		*rig.now = rig.now.Add(time.Minute)
	}
	if _, err = rig.console.Restart(context.Background(), RestartRequest{Workload: "frigate"}); userMessage(t, err) == "" {
		t.Fatal("4th restart within an hour should be refused")
	}
	*rig.now = rig.now.Add(61 * time.Minute)
	if _, err = rig.console.Restart(context.Background(), RestartRequest{Workload: "frigate"}); err != nil {
		t.Fatal(err)
	}
}
