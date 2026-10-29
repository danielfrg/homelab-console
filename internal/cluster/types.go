// Package cluster reads Argo CD apps, workloads, pods, and nodes into a compact
// snapshot for the console UI and agent tools, and performs guarded restarts.
package cluster

import (
	"time"

	"github.com/danielfrg/homelab-launcher/internal/apps"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	restartAnnotation = "kubectl.kubernetes.io/restartedAt"
	optOutAnnotation  = "homelab-console/restart"
)

// Raw is everything read from the cluster before it is joined.
// ArgoErr and MetricsErr record partial-source failures, not fatal ones.
type Raw struct {
	Applications []unstructured.Unstructured
	Deployments  []appsv1.Deployment
	StatefulSets []appsv1.StatefulSet
	Pods         []corev1.Pod
	Nodes        []corev1.Node
	Events       []corev1.Event
	NodeUsage    map[string]NodeUsage
	Links        []apps.App
	ArgoErr      error
	MetricsErr   error
}

type NodeUsage struct {
	CPUMilli    int64
	MemoryBytes int64
}

type Snapshot struct {
	UpdatedAt        time.Time  `json:"updated_at"`
	Stale            bool       `json:"stale"`
	ArgoAvailable    bool       `json:"argo_available"`
	MetricsAvailable bool       `json:"metrics_available"`
	Apps             []App      `json:"apps"`
	Workloads        []Workload `json:"workloads"`
	Pods             []Pod      `json:"pods"`
	Nodes            []Node     `json:"nodes"`
	Events           []Event    `json:"events"`
}

type App struct {
	Name                string     `json:"name"`
	Project             string     `json:"project"`
	Namespace           string     `json:"namespace"`
	Sync                string     `json:"sync"`
	Health              string     `json:"health"`
	OperationPhase      string     `json:"operation_phase"`
	OperationMessage    string     `json:"operation_message"`
	OperationFinishedAt *time.Time `json:"operation_finished_at"`
	Revision            string     `json:"revision"`
	DeployedAt          *time.Time `json:"deployed_at"`
	Images              []string   `json:"images"`
	Source              string     `json:"source"`
	Links               []Link     `json:"links"`
	Workloads           []string   `json:"workloads" doc:"namespace/name"`
	RestartOptOut       bool       `json:"-"`
}

type Link struct {
	Name    string   `json:"name"`
	URL     string   `json:"url"`
	Aliases []string `json:"aliases"`
}

type Workload struct {
	App                  string     `json:"app"`
	Kind                 string     `json:"kind"`
	Namespace            string     `json:"namespace"`
	Name                 string     `json:"name"`
	Strategy             string     `json:"strategy"`
	Rollout              string     `json:"rollout" doc:"complete, progressing, or stuck"`
	Desired              int32      `json:"desired"`
	Ready                int32      `json:"ready"`
	Updated              int32      `json:"updated"`
	RestartedAt          *time.Time `json:"restarted_at"`
	Restartable          bool       `json:"restartable"`
	NotRestartableReason string     `json:"not_restartable_reason,omitempty"`
	Pods                 []string   `json:"pods"`

	selector   labels.Selector
	inProgress bool
}

type Pod struct {
	Namespace       string       `json:"namespace"`
	Name            string       `json:"name"`
	App             string       `json:"app"`
	Workload        string       `json:"workload"`
	Phase           string       `json:"phase"`
	Node            string       `json:"node"`
	Waiting         string       `json:"waiting,omitempty"`
	Problem         string       `json:"problem,omitempty"`
	Ready           bool         `json:"ready"`
	Restarts        int32        `json:"restarts"`
	LastTermination *Termination `json:"last_termination,omitempty"`
	StartedAt       *time.Time   `json:"started_at"`
	Images          []string     `json:"images"`
	Containers      []string     `json:"containers"`
}

type Termination struct {
	Reason     string    `json:"reason"`
	ExitCode   int32     `json:"exit_code"`
	FinishedAt time.Time `json:"finished_at"`
}

type Node struct {
	Name           string   `json:"name"`
	Role           string   `json:"role"`
	KubeletVersion string   `json:"kubelet_version"`
	KernelVersion  string   `json:"kernel_version"`
	Ready          bool     `json:"ready"`
	Pressure       []string `json:"pressure"`
	CPUPercent     *float64 `json:"cpu_percent"`
	MemoryPercent  *float64 `json:"memory_percent"`
}

type Event struct {
	Namespace string    `json:"namespace"`
	Object    string    `json:"object" doc:"Kind/name"`
	Reason    string    `json:"reason"`
	Message   string    `json:"message"`
	Count     int32     `json:"count"`
	LastSeen  time.Time `json:"last_seen"`
}
