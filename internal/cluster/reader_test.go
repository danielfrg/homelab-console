package cluster

import (
	"context"
	"errors"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func fakeDynamic(objects ...runtime.Object) *dynamicfake.FakeDynamicClient {
	return dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		applicationsGVR: "ApplicationList", nodeMetricsGVR: "NodeMetricsList",
	}, objects...)
}

func nodeMetrics(name, cpu, memory string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "metrics.k8s.io/v1beta1", "kind": "NodeMetrics",
		"metadata": map[string]any{"name": name},
		"usage":    map[string]any{"cpu": cpu, "memory": memory},
	}}
}

func fakeTyped() *fake.Clientset {
	raw := fixture()
	objects := []runtime.Object{}
	for i := range raw.Deployments {
		objects = append(objects, &raw.Deployments[i])
	}
	for i := range raw.StatefulSets {
		objects = append(objects, &raw.StatefulSets[i])
	}
	for i := range raw.Pods {
		objects = append(objects, &raw.Pods[i])
	}
	for i := range raw.Nodes {
		objects = append(objects, &raw.Nodes[i])
	}
	objects = append(objects, &raw.Events[0], &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "normal.1"}, Type: corev1.EventTypeNormal, Reason: "Pulled",
	})
	return fake.NewSimpleClientset(objects...)
}

func TestReaderListsSources(t *testing.T) {
	app := application("sofigram", "Synced", "Healthy", nil, resourceRef{"Deployment", "default", "sofigram"})
	dyn := fakeDynamic(&app)
	if err := dyn.Tracker().Create(nodeMetricsGVR, nodeMetrics("srv-nuc-01", "500m", "4Gi"), ""); err != nil {
		t.Fatal(err)
	}
	reader := newKubeReader(fakeTyped(), dyn)
	raw, err := reader.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(raw.Applications) != 1 || len(raw.Deployments) != 5 || len(raw.StatefulSets) != 1 || len(raw.Pods) != 6 || len(raw.Nodes) != 2 {
		t.Fatalf("raw %d %d %d %d %d", len(raw.Applications), len(raw.Deployments), len(raw.StatefulSets), len(raw.Pods), len(raw.Nodes))
	}
	if len(raw.Events) != 1 || raw.Events[0].Type != corev1.EventTypeWarning {
		t.Fatalf("events %+v", raw.Events)
	}
	if u := raw.NodeUsage["srv-nuc-01"]; u.CPUMilli != 500 || u.MemoryBytes != 4<<30 || raw.MetricsErr != nil || raw.ArgoErr != nil {
		t.Fatalf("usage %+v %v", raw.NodeUsage, raw.MetricsErr)
	}
}

func TestReaderToleratesMissingMetrics(t *testing.T) {
	dyn := fakeDynamic()
	dyn.PrependReactor("list", "nodes", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("the server could not find the requested resource")
	})
	dyn.PrependReactor("list", "applications", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("no argo")
	})
	raw, err := newKubeReader(fakeTyped(), dyn).Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if raw.MetricsErr == nil || raw.NodeUsage != nil || raw.ArgoErr == nil || len(raw.Pods) != 6 {
		t.Fatalf("raw %+v", raw)
	}
	s := Build(raw, testNow)
	if s.MetricsAvailable || s.ArgoAvailable || len(s.Nodes) != 2 {
		t.Fatalf("snapshot %+v", s)
	}
}

func TestReaderFailsWhenPodsFail(t *testing.T) {
	typed := fakeTyped()
	typed.PrependReactor("list", "pods", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("boom")
	})
	if _, err := newKubeReader(typed, fakeDynamic()).Read(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}
