package cluster

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	maxRestartsPerHour = 3
	restartMessage     = "Rollout started; check get_app for progress."
)

type RestartRequest struct {
	Workload string
	App      string
	Reason   string
	Source   string // "mcp" or "ui"
}

type RestartedWorkload struct {
	Kind        string    `json:"kind"`
	Namespace   string    `json:"namespace"`
	Name        string    `json:"name"`
	RestartedAt time.Time `json:"restarted_at"`
}

type RestartResult struct {
	Restarted []RestartedWorkload `json:"restarted"`
	Skipped   []string            `json:"skipped,omitempty"`
	Message   string              `json:"message"`
}

// Restart performs a rolling restart exactly like `kubectl rollout restart`.
// Writes are strict: names are never reinterpreted between apps and workloads.
func (c *Console) Restart(ctx context.Context, req RestartRequest) (RestartResult, error) {
	if !c.restartEnabled {
		return RestartResult{}, userErrorf("restart is disabled")
	}
	workload, app := strings.TrimSpace(req.Workload), strings.TrimSpace(req.App)
	if (workload == "") == (app == "") {
		return RestartResult{}, userErrorf("pass exactly one of workload or app")
	}
	snap, err := c.inv.Snapshot(ctx, true)
	if err != nil {
		return RestartResult{}, err
	}
	if snap.Stale {
		return RestartResult{}, userErrorf("cluster data is stale because the Kubernetes API did not answer; not restarting anything")
	}
	targets, skipped, err := restartTargets(snap, workload, app)
	if err != nil {
		return RestartResult{}, err
	}
	now := c.now().UTC()
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, w := range targets {
		if w.inProgress {
			return RestartResult{}, userErrorf("rollout in progress for " + w.Namespace + "/" + w.Name + "; check get_app")
		}
		if len(c.recent(w, now)) >= maxRestartsPerHour {
			return RestartResult{}, userErrorf(fmt.Sprintf("%s/%s was restarted %d times in the last hour; check get_app and get_logs instead", w.Namespace, w.Name, maxRestartsPerHour))
		}
	}
	out := RestartResult{Restarted: []RestartedWorkload{}, Skipped: skipped, Message: restartMessage}
	stamp := now.Format(time.RFC3339)
	patch, _ := json.Marshal(map[string]any{"spec": map[string]any{"template": map[string]any{"metadata": map[string]any{
		"annotations": map[string]string{restartAnnotation: stamp},
	}}}})
	for _, w := range targets {
		if err := c.patch(ctx, w, patch); err != nil {
			c.inv.Invalidate()
			return out, userErrorf(partialMessage(out.Restarted, w))
		}
		key := w.Namespace + "/" + w.Name
		c.restarts[key] = append(c.recent(w, now), now)
		c.recordEvent(ctx, w, req, now)
		out.Restarted = append(out.Restarted, RestartedWorkload{Kind: w.Kind, Namespace: w.Namespace, Name: w.Name, RestartedAt: now})
	}
	c.inv.Invalidate()
	return out, nil
}

// partialMessage says exactly what happened so a retry does not re-roll
// workloads that already restarted. Raw API errors stay out of it.
func partialMessage(done []RestartedWorkload, failed Workload) string {
	message := "could not restart " + failed.Namespace + "/" + failed.Name + ": the Kubernetes API refused the change"
	if len(done) == 0 {
		return message
	}
	names := []string{}
	for _, d := range done {
		names = append(names, d.Namespace+"/"+d.Name)
	}
	return "restarted " + strings.Join(names, ", ") + "; " + message + "; check get_app before retrying"
}

func restartTargets(snap Snapshot, workload, app string) ([]Workload, []string, error) {
	if workload != "" {
		w, err := snap.FindWorkload(workload)
		var ambiguous *AmbiguousError
		switch {
		case errors.Is(err, ErrNotFound):
			return nil, nil, userErrorf("no workload named " + workload)
		case errors.As(err, &ambiguous):
			return nil, nil, userErrorf(ambiguous.Error() + "; pass namespace/name")
		case err != nil:
			return nil, nil, err
		case !w.Restartable:
			return nil, nil, userErrorf(w.Namespace + "/" + w.Name + " cannot be restarted: " + w.NotRestartableReason)
		}
		return []Workload{w}, nil, nil
	}
	var found *App
	for i := range snap.Apps {
		if strings.EqualFold(snap.Apps[i].Name, app) {
			found = &snap.Apps[i]
		}
	}
	if found == nil {
		if w, err := snap.FindWorkload(app); err == nil && w.App != "" {
			return nil, nil, userErrorf(w.Name + " is a workload in app " + w.App + "; pass workload")
		}
		return nil, nil, userErrorf("no app named " + app)
	}
	targets, skipped := []Workload{}, []string{}
	for _, key := range found.Workloads {
		w, err := snap.FindWorkload(key)
		if err != nil {
			continue
		}
		if w.Restartable {
			targets = append(targets, w)
		} else {
			skipped = append(skipped, key+": "+w.NotRestartableReason)
		}
	}
	if len(targets) == 0 {
		if len(skipped) > 0 {
			return nil, nil, userErrorf("nothing in " + found.Name + " can be restarted: " + strings.Join(skipped, "; "))
		}
		return nil, nil, userErrorf(found.Name + " has no workloads to restart")
	}
	return targets, skipped, nil
}

// recent returns restart times within the last hour; the caller holds c.mu.
func (c *Console) recent(w Workload, now time.Time) []time.Time {
	kept := []time.Time{}
	for _, t := range c.restarts[w.Namespace+"/"+w.Name] {
		if now.Sub(t) < time.Hour {
			kept = append(kept, t)
		}
	}
	return kept
}

func (c *Console) patch(ctx context.Context, w Workload, patch []byte) error {
	var err error
	switch w.Kind {
	case "Deployment":
		_, err = c.typed.AppsV1().Deployments(w.Namespace).Patch(ctx, w.Name, types.StrategicMergePatchType, patch, metav1.PatchOptions{FieldManager: "homelab-console"})
	case "StatefulSet":
		_, err = c.typed.AppsV1().StatefulSets(w.Namespace).Patch(ctx, w.Name, types.StrategicMergePatchType, patch, metav1.PatchOptions{FieldManager: "homelab-console"})
	default:
		err = fmt.Errorf("unsupported kind %s", w.Kind)
	}
	return err
}

// recordEvent is best effort: a missing event must not undo a restart that happened.
func (c *Console) recordEvent(ctx context.Context, w Workload, req RestartRequest, now time.Time) {
	source := req.Source
	if source == "" {
		source = "api"
	}
	message := "Restarted by homelab-console via " + source
	if reason := strings.TrimSpace(req.Reason); reason != "" {
		message += ": " + reason
	}
	stamp := metav1.NewTime(now)
	_, _ = c.typed.CoreV1().Events(w.Namespace).Create(ctx, &corev1.Event{
		ObjectMeta:     metav1.ObjectMeta{GenerateName: w.Name + ".", Namespace: w.Namespace},
		InvolvedObject: corev1.ObjectReference{APIVersion: "apps/v1", Kind: w.Kind, Namespace: w.Namespace, Name: w.Name},
		Reason:         "Restarted", Message: message, Type: corev1.EventTypeNormal, Count: 1,
		Source:         corev1.EventSource{Component: "homelab-console"},
		FirstTimestamp: stamp, LastTimestamp: stamp,
	}, metav1.CreateOptions{})
}
