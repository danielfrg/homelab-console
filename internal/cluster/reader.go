package cluster

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	requestTimeout = 5 * time.Second // matches the launcher catalog
	argoNamespace  = "argocd"
)

var (
	applicationsGVR = schema.GroupVersionResource{Group: "argoproj.io", Version: "v1alpha1", Resource: "applications"}
	nodeMetricsGVR  = schema.GroupVersionResource{Group: "metrics.k8s.io", Version: "v1beta1", Resource: "nodes"}
)

type Reader interface {
	Read(ctx context.Context) (Raw, error)
}

// KubeReader lists cluster state. Workloads, pods, and nodes are required;
// Argo CD and metrics-server failures are recorded in Raw and do not fail the read.
type KubeReader struct {
	typed   kubernetes.Interface
	dynamic dynamic.Interface
}

// BoundedConfig copies config with the request timeout and user agent every
// cluster call uses, including restart patches and log reads.
func BoundedConfig(config *rest.Config) *rest.Config {
	bounded := rest.CopyConfig(config)
	bounded.Timeout = requestTimeout
	bounded.UserAgent = "homelab-console"
	return bounded
}

func NewKubeReader(config *rest.Config) (*KubeReader, error) {
	bounded := BoundedConfig(config)
	typed, err := kubernetes.NewForConfig(bounded)
	if err != nil {
		return nil, err
	}
	dyn, err := dynamic.NewForConfig(bounded)
	if err != nil {
		return nil, err
	}
	return newKubeReader(typed, dyn), nil
}

func newKubeReader(typed kubernetes.Interface, dyn dynamic.Interface) *KubeReader {
	return &KubeReader{typed: typed, dynamic: dyn}
}

func (r *KubeReader) Read(ctx context.Context) (Raw, error) {
	all := metav1.ListOptions{}
	raw := Raw{}
	deployments, err := r.typed.AppsV1().Deployments("").List(ctx, all)
	if err != nil {
		return Raw{}, fmt.Errorf("list deployments: %w", err)
	}
	raw.Deployments = deployments.Items
	statefulSets, err := r.typed.AppsV1().StatefulSets("").List(ctx, all)
	if err != nil {
		return Raw{}, fmt.Errorf("list statefulsets: %w", err)
	}
	raw.StatefulSets = statefulSets.Items
	pods, err := r.typed.CoreV1().Pods("").List(ctx, all)
	if err != nil {
		return Raw{}, fmt.Errorf("list pods: %w", err)
	}
	raw.Pods = pods.Items
	nodes, err := r.typed.CoreV1().Nodes().List(ctx, all)
	if err != nil {
		return Raw{}, fmt.Errorf("list nodes: %w", err)
	}
	raw.Nodes = nodes.Items
	if events, err := r.typed.CoreV1().Events("").List(ctx, metav1.ListOptions{FieldSelector: "type=Warning"}); err == nil {
		for _, e := range events.Items {
			// The fake client ignores field selectors; filter here as well.
			if e.Type == corev1.EventTypeWarning {
				raw.Events = append(raw.Events, e)
			}
		}
	}
	if applications, err := r.dynamic.Resource(applicationsGVR).Namespace(argoNamespace).List(ctx, all); err != nil {
		raw.ArgoErr = err
	} else {
		raw.Applications = applications.Items
	}
	if metrics, err := r.dynamic.Resource(nodeMetricsGVR).List(ctx, all); err != nil {
		raw.MetricsErr = err
	} else {
		raw.NodeUsage = map[string]NodeUsage{}
		for _, item := range metrics.Items {
			raw.NodeUsage[item.GetName()] = nodeUsage(item)
		}
	}
	return raw, nil
}

func nodeUsage(item unstructured.Unstructured) NodeUsage {
	usage := NodeUsage{}
	if cpu, _, _ := unstructured.NestedString(item.Object, "usage", "cpu"); cpu != "" {
		if q, err := resource.ParseQuantity(cpu); err == nil {
			usage.CPUMilli = q.MilliValue()
		}
	}
	if memory, _, _ := unstructured.NestedString(item.Object, "usage", "memory"); memory != "" {
		if q, err := resource.ParseQuantity(memory); err == nil {
			usage.MemoryBytes = q.Value()
		}
	}
	return usage
}
