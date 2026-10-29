package cluster

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"io"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	k8stesting "k8s.io/client-go/testing"
)

// A broad selector on one Deployment must not claim another Deployment's pods.
func TestBuildMatchesPodsByOwner(t *testing.T) {
	raw := fixture()
	broad := deployment("default", "aaa-broad", 1, 1, 1)
	broad.Spec.Selector = &metav1.LabelSelector{MatchLabels: map[string]string{"app.kubernetes.io/part-of": "ha"}}
	raw.Deployments = append(raw.Deployments, broad)
	p := pod("default", "frigate-6d4f-abc", "frigate", testNow.Add(-time.Hour))
	p.Labels["app.kubernetes.io/part-of"] = "ha"
	p.Labels["pod-template-hash"] = "6d4f"
	p.OwnerReferences = []metav1.OwnerReference{{Kind: "ReplicaSet", Name: "frigate-6d4f", APIVersion: "apps/v1"}}
	vault := pod("vault", "vault-0", "vault", testNow.Add(-time.Hour))
	vault.Labels["app.kubernetes.io/part-of"] = "ha"
	vault.OwnerReferences = []metav1.OwnerReference{{Kind: "StatefulSet", Name: "vault", APIVersion: "apps/v1"}}
	raw.Pods = []corev1.Pod{p, vault}
	s := Build(raw, testNow)
	if got := findPod(t, s, "default", "frigate-6d4f-abc"); got.Workload != "frigate" || got.App != "home-assistant" {
		t.Fatalf("pod matched %q", got.Workload)
	}
	if got := findPod(t, s, "vault", "vault-0"); got.Workload != "vault" {
		t.Fatalf("statefulset pod matched %q", got.Workload)
	}
}

func TestLogsPreviousIsPerContainer(t *testing.T) {
	rig := newRig(t, true, func(raw *Raw) {
		p := &raw.Pods[0] // frigate-abc
		p.Spec.Containers = append(p.Spec.Containers, corev1.Container{Name: "sidecar"})
		p.Status.ContainerStatuses = append(p.Status.ContainerStatuses, corev1.ContainerStatus{
			Name: "sidecar", RestartCount: 1,
			LastTerminationState: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Reason: "Error", FinishedAt: metav1.NewTime(testNow.Add(-time.Hour))}},
		})
	})
	ctx := context.Background()
	_, err := rig.console.Logs(ctx, LogsRequest{Pod: "frigate-abc", Container: "frigate", Previous: true})
	if !strings.Contains(userMessage(t, err), "no previous container logs") {
		t.Fatal(err)
	}
	if _, err = rig.console.Logs(ctx, LogsRequest{Pod: "frigate-abc", Container: "sidecar", Previous: true}); err != nil {
		t.Fatal(err)
	}
	rig.console.logStream = func(context.Context, string, string, *corev1.PodLogOptions) (io.ReadCloser, error) {
		return nil, apierrors.NewBadRequest("previous terminated container not found")
	}
	_, err = rig.console.Logs(ctx, LogsRequest{Pod: "frigate-abc", Container: "sidecar", Previous: true})
	if !strings.Contains(userMessage(t, err), "no previous container logs") {
		t.Fatal(err)
	}
}

func TestRestartPartialFailureIsReported(t *testing.T) {
	rig := newRig(t, true, nil)
	rig.typed.PrependReactor("patch", "deployments", func(a k8stesting.Action) (bool, runtime.Object, error) {
		if a.(k8stesting.PatchAction).GetName() == "home-assistant" {
			return true, nil, apierrors.NewForbidden(schemaGR(), "home-assistant", errors.New("rbac secret detail"))
		}
		return false, nil, nil
	})
	_, err := rig.console.Restart(context.Background(), RestartRequest{App: "home-assistant"})
	msg := userMessage(t, err)
	if !strings.Contains(msg, "default/frigate") || !strings.Contains(msg, "default/home-assistant") || strings.Contains(msg, "secret") {
		t.Fatal(msg)
	}
}

func TestRestartRefusesStaleSnapshot(t *testing.T) {
	rig := newRig(t, true, nil)
	if _, err := rig.console.Snapshot(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	inv := rig.console.inv
	inv.reader = &stubReader{err: errors.New("api down")}
	_, err := rig.console.Restart(context.Background(), RestartRequest{Workload: "frigate"})
	if !strings.Contains(userMessage(t, err), "stale") {
		t.Fatal(err)
	}
}

func TestBoundedConfig(t *testing.T) {
	c := BoundedConfig(&rest.Config{Host: "https://k8s"})
	if c.Timeout != requestTimeout || c.UserAgent != "homelab-console" {
		t.Fatalf("%+v", c)
	}
}
