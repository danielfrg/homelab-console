package cluster

import (
	"context"
	"errors"
	"io"
	"slices"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	defaultLogLines = 100
	maxLogLines     = 500
	maxLogBytes     = 64 << 10
)

type LogsRequest struct {
	Workload  string
	Pod       string
	Container string
	Previous  bool
	Lines     int
}

type LogsResult struct {
	Pod       string   `json:"pod"`
	Container string   `json:"container"`
	OtherPods []string `json:"other_pods"`
	Text      string   `json:"text"`
	Truncated bool     `json:"truncated"`
}

// Logs reads a bounded tail of one pod's logs live from the Kubernetes API.
func (c *Console) Logs(ctx context.Context, req LogsRequest) (LogsResult, error) {
	lines := req.Lines
	if lines == 0 {
		lines = defaultLogLines
	}
	if lines < 1 || lines > maxLogLines {
		return LogsResult{}, userErrorf("lines must be 1 to 500")
	}
	snap, err := c.inv.Snapshot(ctx, false)
	if err != nil {
		return LogsResult{}, err
	}
	target, others, err := logTarget(snap, req)
	if err != nil {
		return LogsResult{}, err
	}
	container := strings.TrimSpace(req.Container)
	if container == "" && len(target.Containers) > 0 {
		container = target.Containers[0]
	}
	if container != "" && len(target.Containers) > 0 && !slices.Contains(target.Containers, container) {
		return LogsResult{}, userErrorf("pod " + target.Name + " has containers: " + strings.Join(target.Containers, ", "))
	}
	noPrevious := userErrorf("no previous container logs for " + target.Name + "/" + container + "; it has not restarted")
	if req.Previous && !slices.Contains(target.terminated, container) {
		return LogsResult{}, noPrevious
	}
	tail := int64(lines)
	stream, err := c.logStream(ctx, target.Namespace, target.Name, &corev1.PodLogOptions{Container: container, Previous: req.Previous, TailLines: &tail})
	if err != nil {
		if req.Previous && apierrors.IsBadRequest(err) {
			return LogsResult{}, noPrevious
		}
		return LogsResult{}, err
	}
	defer stream.Close()
	body, err := io.ReadAll(io.LimitReader(stream, maxLogBytes+1))
	if err != nil {
		return LogsResult{}, err
	}
	out := LogsResult{Pod: target.Name, Container: container, OtherPods: others}
	if len(body) > maxLogBytes {
		body, out.Truncated = body[:maxLogBytes], true
	}
	out.Text = string(body)
	return out, nil
}

func logTarget(snap Snapshot, req LogsRequest) (Pod, []string, error) {
	switch {
	case strings.TrimSpace(req.Pod) != "":
		return lookupPod(snap, req.Pod)
	case strings.TrimSpace(req.Workload) != "":
		w, err := snap.FindWorkload(req.Workload)
		if errors.Is(err, ErrNotFound) {
			return Pod{}, nil, userErrorf("no workload named " + req.Workload)
		}
		if err != nil {
			return Pod{}, nil, userErrorf(err.Error())
		}
		pods := []Pod{}
		for _, p := range snap.Pods {
			if p.Namespace == w.Namespace && p.Workload == w.Name {
				pods = append(pods, p)
			}
		}
		if len(pods) == 0 {
			return Pod{}, nil, userErrorf(w.Namespace + "/" + w.Name + " has no pods")
		}
		sort.SliceStable(pods, func(i, j int) bool { return started(pods[i]) > started(pods[j]) })
		others := []string{}
		for _, p := range pods[1:] {
			others = append(others, p.Name)
		}
		return pods[0], others, nil
	default:
		return Pod{}, nil, userErrorf("pass workload or pod")
	}
}

func started(p Pod) int64 {
	if p.StartedAt == nil {
		return 0
	}
	return p.StartedAt.UnixNano()
}

func lookupPod(snap Snapshot, ref string) (Pod, []string, error) {
	ref = strings.TrimSpace(ref)
	namespace, name, qualified := strings.Cut(ref, "/")
	if !qualified {
		name, namespace = namespace, ""
	}
	matches := []Pod{}
	for _, p := range snap.Pods {
		if p.Name == name && (namespace == "" || p.Namespace == namespace) {
			matches = append(matches, p)
		}
	}
	switch len(matches) {
	case 0:
		return Pod{}, nil, userErrorf("no pod named " + ref)
	case 1:
		return matches[0], []string{}, nil
	}
	return Pod{}, nil, userErrorf("several pods are named " + name + "; pass namespace/name")
}
