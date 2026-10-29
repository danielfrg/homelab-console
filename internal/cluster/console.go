package cluster

import (
	"context"
	"io"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

// UserError carries a message that is safe to show to people and agents.
// Every other error is replaced with a generic message at the API edges.
type UserError struct{ Message string }

func (e *UserError) Error() string { return e.Message }

func userErrorf(message string) error { return &UserError{Message: message} }

// Console is the single entry point for the HTTP API and MCP tools.
type Console struct {
	inv            *Inventory
	typed          kubernetes.Interface
	restartEnabled bool
	now            func() time.Time
	logStream      func(ctx context.Context, namespace, pod string, opts *corev1.PodLogOptions) (io.ReadCloser, error)

	mu       sync.Mutex
	restarts map[string][]time.Time // "namespace/name" -> recent restart times
}

func NewConsole(inv *Inventory, typed kubernetes.Interface, restartEnabled bool, now func() time.Time) *Console {
	c := &Console{inv: inv, typed: typed, restartEnabled: restartEnabled, now: now, restarts: map[string][]time.Time{}}
	c.logStream = func(ctx context.Context, namespace, pod string, opts *corev1.PodLogOptions) (io.ReadCloser, error) {
		return typed.CoreV1().Pods(namespace).GetLogs(pod, opts).Stream(ctx)
	}
	return c
}

func (c *Console) Snapshot(ctx context.Context, refresh bool) (Snapshot, error) {
	return c.inv.Snapshot(ctx, refresh)
}

func (c *Console) RestartEnabled() bool { return c.restartEnabled }
