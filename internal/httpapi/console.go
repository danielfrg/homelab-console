package httpapi

import (
	"context"
	"errors"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/danielfrg/homelab-launcher/internal/cluster"
	"github.com/danielfrg/homelab-launcher/internal/mcp"
	"github.com/danielgtaylor/huma/v2"
)

type ConfigOutput struct {
	Body struct {
		RestartEnabled bool `json:"restart_enabled"`
	}
}

type OverviewOutput struct{ Body cluster.Overview }

type AppListInput struct {
	Query  string `query:"query"`
	Status string `query:"status" enum:"all,problems" default:"all"`
}
type AppListOutput struct {
	Body struct {
		Apps      []cluster.AppSummary `json:"apps" nullable:"false"`
		UpdatedAt time.Time            `json:"updated_at"`
		Stale     bool                 `json:"stale"`
	}
}

type AppInput struct {
	Name string `path:"name"`
}
type AppOutput struct{ Body mcp.GetAppOutput }

type PodsInput struct {
	App          string `query:"app"`
	Namespace    string `query:"namespace"`
	Node         string `query:"node"`
	ProblemsOnly bool   `query:"problems_only"`
}
type PodsOutput struct {
	Body struct {
		Pods      []cluster.Pod `json:"pods" nullable:"false"`
		UpdatedAt time.Time     `json:"updated_at"`
		Stale     bool          `json:"stale"`
	}
}

type LogsInput struct {
	Workload  string `query:"workload"`
	Pod       string `query:"pod"`
	Container string `query:"container"`
	Previous  bool   `query:"previous"`
	Lines     int    `query:"lines"`
}
type LogsOutput struct{ Body cluster.LogsResult }

type RestartInput struct {
	Body struct {
		Workload string `json:"workload,omitempty"`
		App      string `json:"app,omitempty"`
		Reason   string `json:"reason,omitempty"`
	}
}
type RestartOutput struct{ Body cluster.RestartResult }

func registerConsole(api huma.API, console mcp.Console) {
	snapshot := func(ctx context.Context) (cluster.Snapshot, error) {
		if console == nil {
			return cluster.Snapshot{}, unavailable()
		}
		v, err := console.Snapshot(ctx, false)
		if err != nil {
			return cluster.Snapshot{}, consoleError(err)
		}
		return v, nil
	}
	huma.Register(api, huma.Operation{OperationID: "get-config", Method: http.MethodGet, Path: "/api/config", Summary: "Get console settings"},
		func(_ context.Context, _ *struct{}) (*ConfigOutput, error) {
			out := &ConfigOutput{}
			out.Body.RestartEnabled = console != nil && console.RestartEnabled()
			return out, nil
		})
	huma.Register(api, huma.Operation{OperationID: "get-overview", Method: http.MethodGet, Path: "/api/overview", Summary: "Summarize cluster health", Errors: []int{503}},
		func(ctx context.Context, _ *struct{}) (*OverviewOutput, error) {
			v, err := snapshot(ctx)
			if err != nil {
				return nil, err
			}
			return &OverviewOutput{Body: v.Overview(time.Now())}, nil
		})
	huma.Register(api, huma.Operation{OperationID: "list-apps", Method: http.MethodGet, Path: "/api/apps", Summary: "List Argo CD apps", Errors: []int{503}},
		func(ctx context.Context, in *AppListInput) (*AppListOutput, error) {
			v, err := snapshot(ctx)
			if err != nil {
				return nil, err
			}
			out := &AppListOutput{}
			out.Body.UpdatedAt, out.Body.Stale = v.UpdatedAt, v.Stale
			out.Body.Apps = []cluster.AppSummary{}
			for _, s := range v.Summaries() {
				if in.Status == "problems" && s.Sync == "Synced" && s.Health == "Healthy" {
					continue
				}
				if q := strings.ToLower(strings.TrimSpace(in.Query)); q != "" && !strings.Contains(strings.ToLower(s.Name), q) {
					continue
				}
				out.Body.Apps = append(out.Body.Apps, s)
			}
			return out, nil
		})
	huma.Register(api, huma.Operation{OperationID: "get-app", Method: http.MethodGet, Path: "/api/apps/{name}", Summary: "Get one app with workloads and pods", Errors: []int{404, 503}},
		func(ctx context.Context, in *AppInput) (*AppOutput, error) {
			v, err := snapshot(ctx)
			if err != nil {
				return nil, err
			}
			app, matched, err := v.FindApp(in.Name)
			if err != nil {
				return nil, consoleError(err)
			}
			return &AppOutput{Body: mcp.AppDetail(v, app, matched)}, nil
		})
	huma.Register(api, huma.Operation{OperationID: "list-pods", Method: http.MethodGet, Path: "/api/pods", Summary: "List pods", Errors: []int{503}},
		func(ctx context.Context, in *PodsInput) (*PodsOutput, error) {
			v, err := snapshot(ctx)
			if err != nil {
				return nil, err
			}
			out := &PodsOutput{}
			out.Body.UpdatedAt, out.Body.Stale = v.UpdatedAt, v.Stale
			out.Body.Pods = mcp.FilterPods(v.Pods, in.App, in.Namespace, in.Node, in.ProblemsOnly)
			return out, nil
		})
	huma.Register(api, huma.Operation{OperationID: "get-logs", Method: http.MethodGet, Path: "/api/logs", Summary: "Fetch recent pod logs", Errors: []int{400, 503}},
		func(ctx context.Context, in *LogsInput) (*LogsOutput, error) {
			if console == nil {
				return nil, unavailable()
			}
			out, err := console.Logs(ctx, cluster.LogsRequest{Workload: in.Workload, Pod: in.Pod, Container: in.Container, Previous: in.Previous, Lines: in.Lines})
			if err != nil {
				return nil, consoleError(err)
			}
			return &LogsOutput{Body: out}, nil
		})
	huma.Register(api, huma.Operation{OperationID: "restart", Method: http.MethodPost, Path: "/api/restart", Summary: "Rolling restart of a workload or app", Errors: []int{400, 403, 415, 503}},
		func(ctx context.Context, in *RestartInput) (*RestartOutput, error) {
			if console == nil {
				return nil, unavailable()
			}
			out, err := console.Restart(ctx, cluster.RestartRequest{Workload: in.Body.Workload, App: in.Body.App, Reason: in.Body.Reason, Source: "ui"})
			if err != nil {
				return nil, consoleError(err)
			}
			return &RestartOutput{Body: out}, nil
		})
}

func unavailable() error {
	return huma.Error503ServiceUnavailable("Cluster data is temporarily unavailable")
}

func consoleError(err error) error {
	var userErr *cluster.UserError
	var ambiguous *cluster.AmbiguousError
	switch {
	case errors.As(err, &userErr):
		return huma.Error400BadRequest(userErr.Message)
	case errors.As(err, &ambiguous):
		return huma.Error400BadRequest(ambiguous.Error())
	case errors.Is(err, cluster.ErrNotFound):
		return huma.Error404NotFound("Not found")
	default:
		return unavailable()
	}
}

// allowedHosts guards the endpoints that reveal logs or change the cluster
// against DNS rebinding, using the same host list as MCP.
func allowedHosts(next http.Handler, hosts []string) http.Handler {
	allowed := map[string]bool{"localhost": true, "127.0.0.1": true, "::1": true}
	for _, host := range hosts {
		allowed[strings.ToLower(strings.Trim(host, "[]"))] = true
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/restart" || r.URL.Path == "/api/logs" {
			host := r.Host
			if h, _, err := net.SplitHostPort(host); err == nil {
				host = h
			}
			if !allowed[strings.ToLower(strings.Trim(host, "[]"))] {
				http.Error(w, "Host is not allowed", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// sameOriginWrites rejects browser requests to write endpoints from other
// sites. There is no auth, so this is what stops a page elsewhere from
// triggering a restart. Non-browser clients send no Origin or Sec-Fetch-Site.
func sameOriginWrites(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/restart" {
			next.ServeHTTP(w, r)
			return
		}
		if site := r.Header.Get("Sec-Fetch-Site"); site != "" && site != "same-origin" && site != "none" {
			http.Error(w, "Cross-site requests are not allowed", http.StatusForbidden)
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" {
			u, err := url.Parse(origin)
			if err != nil || !strings.EqualFold(u.Host, r.Host) {
				http.Error(w, "Cross-origin requests are not allowed", http.StatusForbidden)
				return
			}
		}
		if mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mediaType != "application/json" {
			http.Error(w, "Send application/json", http.StatusUnsupportedMediaType)
			return
		}
		next.ServeHTTP(w, r)
	})
}
