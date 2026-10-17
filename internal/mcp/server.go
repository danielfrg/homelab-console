package mcp

import (
	"context"
	"errors"
	"github.com/danielfrg/homelab-launcher/internal/apps"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

type Catalog interface {
	Snapshot(context.Context, bool) (apps.Snapshot, error)
}
type Freshness struct {
	UpdatedAt time.Time `json:"updated_at"`
	Stale     bool      `json:"stale"`
}
type SearchInput struct {
	Paging
	Query string `json:"query,omitempty"`
	Group string `json:"group,omitempty"`
}
type SearchOutput struct {
	Freshness
	Apps       []apps.App `json:"apps"`
	Total      int        `json:"total"`
	NextOffset *int       `json:"next_offset"`
}
type GetInput struct {
	AppID string `json:"app_id"`
}
type GetOutput struct {
	Freshness
	App apps.App `json:"app"`
}

func NewHandler(catalog Catalog, hosts []string) http.Handler {
	server := sdk.NewServer(&sdk.Implementation{Name: "homelab-launcher", Version: "0.1.0"}, nil)
	read := func(ctx context.Context) (apps.Snapshot, error) {
		if catalog == nil {
			return apps.Snapshot{}, errors.New("App discovery unavailable; try again")
		}
		if ctx.Err() != nil {
			return apps.Snapshot{}, errors.New("App request canceled")
		}
		v, e := catalog.Snapshot(ctx, false)
		if e != nil {
			return apps.Snapshot{}, errors.New("App discovery unavailable; try again")
		}
		return v, nil
	}
	normalize := func(app apps.App) apps.App {
		app.Group = strings.TrimSpace(app.Group)
		if app.Group == "" {
			app.Group = "Apps"
		}
		if app.Aliases == nil {
			app.Aliases = []string{}
		}
		return app
	}
	sdk.AddTool(server, &sdk.Tool{Name: "search_apps", Description: "Search the cached route directory by name, description, literal group, hostname or aliases. Results indicate staleness, not destination health.", Annotations: annotation()}, func(ctx context.Context, _ *sdk.CallToolRequest, in SearchInput) (*sdk.CallToolResult, SearchOutput, error) {
		out := SearchOutput{Apps: []apps.App{}}
		if _, _, _, e := in.Paging.bounds(0); e != nil {
			return nil, out, e
		}
		v, e := read(ctx)
		if e != nil {
			return nil, out, e
		}
		out.Freshness = Freshness{v.UpdatedAt, v.Stale}
		items := []apps.App{}
		q := strings.ToLower(strings.TrimSpace(in.Query))
		for _, a := range v.Apps {
			a = normalize(a)
			if in.Group != "" && a.Group != in.Group {
				continue
			}
			u, _ := url.Parse(a.URL)
			host := ""
			if u != nil {
				host = u.Hostname()
			}
			haystack := strings.Join(append([]string{a.Name, a.Description, a.Group, host}, a.Aliases...), " ")
			if strings.Contains(strings.ToLower(haystack), q) {
				items = append(items, a)
			}
		}
		sort.Slice(items, func(i, j int) bool {
			a, b := items[i], items[j]
			if a.Name != b.Name {
				return a.Name < b.Name
			}
			return a.ID < b.ID
		})
		a, b, next, _ := in.Paging.bounds(len(items))
		out.Apps = items[a:b]
		out.Total = len(items)
		out.NextOffset = next
		return nil, out, nil
	})
	sdk.AddTool(server, &sdk.Tool{Name: "get_app", Description: "Get an app's discovered URL and aliases by stable ID. Does not probe the destination or change routes.", Annotations: annotation()}, func(ctx context.Context, _ *sdk.CallToolRequest, in GetInput) (*sdk.CallToolResult, GetOutput, error) {
		v, e := read(ctx)
		if e != nil {
			return nil, GetOutput{}, e
		}
		for _, a := range v.Apps {
			if a.ID == in.AppID {
				return nil, GetOutput{Freshness{v.UpdatedAt, v.Stale}, normalize(a)}, nil
			}
		}
		return nil, GetOutput{}, errors.New("App not found")
	})
	return transport(server, hosts)
}
