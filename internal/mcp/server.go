package mcp

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/danielfrg/homelab-console/internal/cluster"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const maxAppEvents = 20

// Console is what the tools need from the cluster package.
type Console interface {
	Snapshot(context.Context, bool) (cluster.Snapshot, error)
	Logs(context.Context, cluster.LogsRequest) (cluster.LogsResult, error)
	Restart(context.Context, cluster.RestartRequest) (cluster.RestartResult, error)
	RestartEnabled() bool
}

type Freshness struct {
	UpdatedAt time.Time `json:"updated_at"`
	Stale     bool      `json:"stale"`
}

type ListAppsInput struct {
	Paging
	Query  string `json:"query,omitempty" jsonschema:"Matches app names, workload names, hostnames, aliases, and link names"`
	Status string `json:"status,omitempty" jsonschema:"all (default) or problems: apps that are not Synced and Healthy"`
}
type ListAppsOutput struct {
	Freshness
	Apps       []cluster.AppSummary `json:"apps"`
	Total      int                  `json:"total"`
	NextOffset *int                 `json:"next_offset"`
}

type GetAppInput struct {
	Name string `json:"name" jsonschema:"An Argo CD app name, or a workload name such as frigate"`
}
type WorkloadDetail struct {
	cluster.Workload
	PodDetails []cluster.Pod `json:"pods"`
}
type GetAppOutput struct {
	Freshness
	App             cluster.App      `json:"app"`
	MatchedWorkload string           `json:"matched_workload"`
	Workloads       []WorkloadDetail `json:"workloads"`
	Events          []cluster.Event  `json:"events"`
}

type ListPodsInput struct {
	Paging
	App          string `json:"app,omitempty" jsonschema:"Argo CD app name"`
	Namespace    string `json:"namespace,omitempty"`
	Node         string `json:"node,omitempty"`
	ProblemsOnly bool   `json:"problems_only,omitempty" jsonschema:"Only pods that are crashing, pending, not ready, or recently restarted"`
}
type ListPodsOutput struct {
	Freshness
	Pods       []cluster.Pod `json:"pods"`
	Total      int           `json:"total"`
	NextOffset *int          `json:"next_offset"`
}

type GetLogsInput struct {
	Workload  string `json:"workload,omitempty" jsonschema:"Workload name or namespace/name; uses its newest pod"`
	Pod       string `json:"pod,omitempty" jsonschema:"Pod name or namespace/name"`
	Container string `json:"container,omitempty" jsonschema:"Defaults to the first container"`
	Previous  bool   `json:"previous,omitempty" jsonschema:"Logs of the previous, crashed container"`
	Lines     int    `json:"lines,omitempty" jsonschema:"Lines from the end, 1 to 500 (default 100)"`
}

type RestartInput struct {
	Workload string `json:"workload,omitempty" jsonschema:"Workload name or namespace/name, such as frigate"`
	App      string `json:"app,omitempty" jsonschema:"Argo CD app name; restarts every restartable workload in it"`
	Reason   string `json:"reason,omitempty" jsonschema:"Why, in the user's words; recorded in the Kubernetes event"`
}

func NewHandler(console Console, hosts []string) http.Handler {
	server := sdk.NewServer(&sdk.Implementation{Name: "homelab-console", Version: "0.2.0"}, nil)
	read := func(ctx context.Context) (cluster.Snapshot, error) {
		if console == nil {
			return cluster.Snapshot{}, errors.New("Cluster data unavailable; try again")
		}
		v, err := console.Snapshot(ctx, false)
		if err != nil {
			return cluster.Snapshot{}, sanitize(err)
		}
		return v, nil
	}

	sdk.AddTool(server, &sdk.Tool{Name: "get_overview", Annotations: annotation(),
		Description: "Is everything OK? Nodes with usage and conditions, Argo CD app totals, apps that are not Synced and Healthy, problem workloads and pods, and recent restarts. Kubernetes status can look healthy while an app is broken; check logs when the user reports a problem."},
		func(ctx context.Context, _ *sdk.CallToolRequest, _ struct{}) (*sdk.CallToolResult, cluster.Overview, error) {
			v, err := read(ctx)
			if err != nil {
				return nil, cluster.Overview{}, err
			}
			return nil, v.Overview(time.Now()), nil
		})

	sdk.AddTool(server, &sdk.Tool{Name: "list_apps", Annotations: annotation(),
		Description: "List Argo CD apps with sync, health, ready workloads, primary link, and last deploy. Search also matches workload names and hostnames."},
		func(ctx context.Context, _ *sdk.CallToolRequest, in ListAppsInput) (*sdk.CallToolResult, ListAppsOutput, error) {
			out := ListAppsOutput{Apps: []cluster.AppSummary{}}
			if _, _, _, err := in.Paging.bounds(0); err != nil {
				return nil, out, err
			}
			status := strings.ToLower(strings.TrimSpace(in.Status))
			if status != "" && status != "all" && status != "problems" {
				return nil, out, errors.New("status must be all or problems")
			}
			v, err := read(ctx)
			if err != nil {
				return nil, out, err
			}
			out.Freshness = Freshness{v.UpdatedAt, v.Stale}
			summaries := v.Summaries()
			items := []cluster.AppSummary{}
			q := strings.ToLower(strings.TrimSpace(in.Query))
			for i, app := range v.Apps {
				s := summaries[i]
				if status == "problems" && s.Sync == "Synced" && s.Health == "Healthy" {
					continue
				}
				if q == "" || strings.Contains(appHaystack(app), q) {
					items = append(items, s)
				}
			}
			a, b, next, _ := in.Paging.bounds(len(items))
			out.Apps, out.Total, out.NextOffset = items[a:b], len(items), next
			return nil, out, nil
		})

	sdk.AddTool(server, &sdk.Tool{Name: "get_app", Annotations: annotation(),
		Description: "One Argo CD app in full: sync and health, last sync and deploy, images, links, workloads with their pods, and recent warning events. Accepts a workload name such as frigate and returns the app that owns it."},
		func(ctx context.Context, _ *sdk.CallToolRequest, in GetAppInput) (*sdk.CallToolResult, GetAppOutput, error) {
			v, err := read(ctx)
			if err != nil {
				return nil, GetAppOutput{}, err
			}
			app, matched, err := v.FindApp(in.Name)
			if err != nil {
				return nil, GetAppOutput{}, notFoundOr(err, "no app or workload named "+in.Name)
			}
			return nil, appDetail(v, app, matched), nil
		})

	sdk.AddTool(server, &sdk.Tool{Name: "list_pods", Annotations: annotation(),
		Description: "List pods across the cluster, including ones outside Argo CD apps, filtered by app, namespace, node, or problems only."},
		func(ctx context.Context, _ *sdk.CallToolRequest, in ListPodsInput) (*sdk.CallToolResult, ListPodsOutput, error) {
			out := ListPodsOutput{Pods: []cluster.Pod{}}
			if _, _, _, err := in.Paging.bounds(0); err != nil {
				return nil, out, err
			}
			v, err := read(ctx)
			if err != nil {
				return nil, out, err
			}
			out.Freshness = Freshness{v.UpdatedAt, v.Stale}
			items := FilterPods(v.Pods, in.App, in.Namespace, in.Node, in.ProblemsOnly)
			a, b, next, _ := in.Paging.bounds(len(items))
			out.Pods, out.Total, out.NextOffset = items[a:b], len(items), next
			return nil, out, nil
		})

	sdk.AddTool(server, &sdk.Tool{Name: "get_logs", Annotations: annotation(),
		Description: "Fetch recent log lines from a pod (newest pod of a workload by default). Output may contain sensitive data and is sent to the model."},
		func(ctx context.Context, _ *sdk.CallToolRequest, in GetLogsInput) (*sdk.CallToolResult, cluster.LogsResult, error) {
			if console == nil {
				return nil, cluster.LogsResult{}, errors.New("Cluster data unavailable; try again")
			}
			out, err := console.Logs(ctx, cluster.LogsRequest{Workload: in.Workload, Pod: in.Pod, Container: in.Container, Previous: in.Previous, Lines: in.Lines})
			if err != nil {
				return nil, cluster.LogsResult{}, sanitize(err)
			}
			return nil, out, nil
		})

	if console != nil && console.RestartEnabled() {
		sdk.AddTool(server, &sdk.Tool{Name: "restart", Annotations: restartAnnotation(),
			Description: "Rolling restart of one workload, or every workload in an Argo app. Only call when the user explicitly asks to restart something. Prefer `workload` for a single service such as frigate."},
			func(ctx context.Context, _ *sdk.CallToolRequest, in RestartInput) (*sdk.CallToolResult, cluster.RestartResult, error) {
				out, err := console.Restart(ctx, cluster.RestartRequest{Workload: in.Workload, App: in.App, Reason: in.Reason, Source: "mcp"})
				if err != nil {
					return nil, cluster.RestartResult{}, sanitize(err)
				}
				return nil, out, nil
			})
	}
	return transport(server, hosts)
}

func appHaystack(app cluster.App) string {
	parts := []string{app.Name}
	for _, w := range app.Workloads {
		_, name, _ := strings.Cut(w, "/")
		parts = append(parts, name)
	}
	for _, l := range app.Links {
		parts = append(parts, l.Name)
		if u, err := url.Parse(l.URL); err == nil {
			parts = append(parts, u.Hostname())
		}
		parts = append(parts, l.Aliases...)
	}
	return strings.ToLower(strings.Join(parts, " "))
}

// FilterPods applies the list_pods filters; the HTTP API shares it.
func FilterPods(pods []cluster.Pod, app, namespace, node string, problemsOnly bool) []cluster.Pod {
	items := []cluster.Pod{}
	for _, p := range pods {
		if (app != "" && !strings.EqualFold(p.App, app)) || (namespace != "" && p.Namespace != namespace) ||
			(node != "" && p.Node != node) || (problemsOnly && p.Problem == "") {
			continue
		}
		items = append(items, p)
	}
	return items
}

// AppDetail builds the get_app result; the HTTP API shares it.
func AppDetail(v cluster.Snapshot, app cluster.App, matched string) GetAppOutput {
	return appDetail(v, app, matched)
}

func appDetail(v cluster.Snapshot, app cluster.App, matched string) GetAppOutput {
	out := GetAppOutput{Freshness: Freshness{v.UpdatedAt, v.Stale}, App: app, MatchedWorkload: matched, Workloads: []WorkloadDetail{}, Events: []cluster.Event{}}
	names := map[string]bool{}
	workloadNames := []string{}
	for _, key := range app.Workloads {
		w, err := v.FindWorkload(key)
		if err != nil {
			continue
		}
		detail := WorkloadDetail{Workload: w, PodDetails: []cluster.Pod{}}
		for _, p := range v.Pods {
			if p.Namespace == w.Namespace && p.Workload == w.Name {
				detail.PodDetails = append(detail.PodDetails, p)
				names[p.Namespace+"/"+p.Name] = true
			}
		}
		names[w.Namespace+"/"+w.Name] = true
		workloadNames = append(workloadNames, w.Namespace+"/"+w.Name+"-")
		out.Workloads = append(out.Workloads, detail)
	}
	for _, e := range v.Events {
		_, object, _ := strings.Cut(e.Object, "/")
		key := e.Namespace + "/" + object
		related := names[key]
		for _, prefix := range workloadNames {
			related = related || strings.HasPrefix(key, prefix)
		}
		if related && len(out.Events) < maxAppEvents {
			out.Events = append(out.Events, e)
		}
	}
	return out
}

// sanitize lets safe messages through and hides everything else.
func sanitize(err error) error {
	var userErr *cluster.UserError
	if errors.As(err, &userErr) {
		return errors.New(userErr.Message)
	}
	if errors.Is(err, context.Canceled) {
		return errors.New("Request canceled")
	}
	return errors.New("Cluster data unavailable; try again")
}

func notFoundOr(err error, message string) error {
	if errors.Is(err, cluster.ErrNotFound) {
		return errors.New(message)
	}
	var ambiguous *cluster.AmbiguousError
	if errors.As(err, &ambiguous) {
		return errors.New(ambiguous.Error())
	}
	return sanitize(err)
}
