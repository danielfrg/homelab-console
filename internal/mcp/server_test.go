package mcp

import (
	"context"
	"errors"
	"github.com/danielfrg/homelab-launcher/internal/apps"
	"testing"
	"time"
)

type catalog struct {
	value apps.Snapshot
	err   error
	t     *testing.T
}

func (c catalog) Snapshot(ctx context.Context, force bool) (apps.Snapshot, error) {
	if force {
		c.t.Fatal("forced refresh")
	}
	if ctx.Err() != nil {
		return apps.Snapshot{}, ctx.Err()
	}
	return c.value, c.err
}
func TestDirectoryTools(t *testing.T) {
	value := apps.Snapshot{UpdatedAt: time.Now().UTC(), Stale: true, Apps: []apps.App{{ID: "b", Name: "Wiki", URL: "https://wiki.example.net", Description: "Knowledge", Group: "Favorites", Aliases: []string{"docs.example.net"}}, {ID: "a", Name: "Build", URL: "https://ci.example.net"}}}
	s := session(t, NewHandler(catalog{value: value, t: t}, nil), 2)
	for _, query := range []string{" WIKI ", "Knowledge", "Favorites", "wiki.example.net", "docs.example.net"} {
		v := call(t, s, "search_apps", map[string]any{"query": query}, false)
		if v["total"] != float64(1) || v["stale"] != true || v["updated_at"] == nil {
			t.Fatal(v)
		}
	}
	page := call(t, s, "search_apps", map[string]any{"limit": 1}, false)
	if page["apps"].([]any)[0].(map[string]any)["id"] != "a" || page["next_offset"] != float64(1) {
		t.Fatal(page)
	}
	def := call(t, s, "search_apps", map[string]any{"group": "Apps"}, false)
	if def["total"] != float64(1) {
		t.Fatal(def)
	}
	one := call(t, s, "get_app", map[string]any{"app_id": "b"}, false)
	if one["app"].(map[string]any)["name"] != "Wiki" {
		t.Fatal(one)
	}
	call(t, s, "get_app", map[string]any{"app_id": "missing"}, true)
	for _, limit := range []int{0, -1, 101} {
		call(t, s, "search_apps", map[string]any{"limit": limit}, true)
	}
	call(t, s, "search_apps", map[string]any{"offset": -1}, true)
	if value.Apps[0].ID != "b" || value.Apps[1].Group != "" {
		t.Fatal("snapshot mutated")
	}
	bad := session(t, NewHandler(catalog{err: errors.New("secret"), t: t}, nil), 2)
	call(t, bad, "search_apps", map[string]any{}, true)
}
