package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/danielfrg/homelab-console/internal/cluster"
)

type consoleStub struct {
	snapshot cluster.Snapshot
	restart  cluster.RestartRequest
	err      error
}

func (c *consoleStub) Snapshot(context.Context, bool) (cluster.Snapshot, error) {
	return c.snapshot, c.err
}
func (c *consoleStub) Logs(_ context.Context, req cluster.LogsRequest) (cluster.LogsResult, error) {
	if req.Lines > 500 {
		return cluster.LogsResult{}, &cluster.UserError{Message: "lines must be 1 to 500"}
	}
	return cluster.LogsResult{Pod: "frigate-abc", Container: "frigate", OtherPods: []string{}, Text: "ok"}, nil
}
func (c *consoleStub) Restart(_ context.Context, req cluster.RestartRequest) (cluster.RestartResult, error) {
	c.restart = req
	if req.Workload == "vault" {
		return cluster.RestartResult{}, &cluster.UserError{Message: "vault/vault cannot be restarted"}
	}
	if c.err != nil {
		return cluster.RestartResult{}, c.err
	}
	return cluster.RestartResult{Restarted: []cluster.RestartedWorkload{{Kind: "Deployment", Namespace: "default", Name: req.Workload}}, Message: "Rollout started; check get_app for progress."}, nil
}
func (c *consoleStub) RestartEnabled() bool { return true }

func stubSnapshot() cluster.Snapshot {
	return cluster.Snapshot{
		UpdatedAt: time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC),
		Apps: []cluster.App{{Name: "home-assistant", Sync: "Synced", Health: "Healthy", Images: []string{}, Links: []cluster.Link{},
			Workloads: []string{"default/frigate"}}},
		Workloads: []cluster.Workload{{App: "home-assistant", Kind: "Deployment", Namespace: "default", Name: "frigate", Desired: 1, Ready: 1, Pods: []string{"frigate-abc"}}},
		Pods:      []cluster.Pod{{Namespace: "default", Name: "frigate-abc", App: "home-assistant", Workload: "frigate", Images: []string{}, Containers: []string{"frigate"}}},
		Nodes:     []cluster.Node{}, Events: []cluster.Event{},
	}
}

func get(t *testing.T, h http.Handler, path string) (int, map[string]any) {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "http://localhost"+path, nil))
	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	return w.Code, body
}

func TestConsoleEndpoints(t *testing.T) {
	h := NewHandler(nil, nil, &consoleStub{snapshot: stubSnapshot()})
	for path, key := range map[string]string{
		"/api/config": "restart_enabled", "/api/overview": "problems", "/api/apps": "apps",
		"/api/apps?status=problems": "apps", "/api/pods?problems_only=true": "pods", "/api/logs?workload=frigate": "text",
	} {
		if code, body := get(t, h, path); code != 200 || body[key] == nil {
			t.Fatalf("%s: %d %v", path, code, body)
		}
	}
	if code, body := get(t, h, "/api/apps/frigate"); code != 200 || body["matched_workload"] != "frigate" {
		t.Fatal(code, body)
	}
	if code, _ := get(t, h, "/api/apps/nope"); code != 404 {
		t.Fatal(code)
	}
	if code, body := get(t, h, "/api/logs?pod=x&lines=900"); code != 400 || !strings.Contains(body["detail"].(string), "lines") {
		t.Fatal(code, body)
	}
	if code, _ := get(t, NewHandler(nil, nil, &consoleStub{err: errors.New("secret")}), "/api/overview"); code != 503 {
		t.Fatal(code)
	}
}

func postRestart(h http.Handler, body string, headers map[string]string) *httptest.ResponseRecorder {
	r := httptest.NewRequest("POST", "http://apps.corp.danielfrg.net/api/restart", strings.NewReader(body))
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestRestartRejectsCrossOrigin(t *testing.T) {
	console := &consoleStub{snapshot: stubSnapshot()}
	h := NewHandler(nil, nil, console, "apps.corp.danielfrg.net")
	body := `{"workload":"frigate","reason":"frozen"}`
	json := map[string]string{"Content-Type": "application/json"}
	cases := []struct {
		headers map[string]string
		status  int
	}{
		{map[string]string{"Content-Type": "application/json", "Sec-Fetch-Site": "cross-site"}, 403},
		{map[string]string{"Content-Type": "application/json", "Origin": "https://evil.example"}, 403},
		{map[string]string{"Content-Type": "text/plain", "Origin": "http://apps.corp.danielfrg.net"}, 415},
		{map[string]string{"Content-Type": "application/json", "Origin": "http://apps.corp.danielfrg.net", "Sec-Fetch-Site": "same-origin"}, 200},
		{json, 200},
	}
	for i, tc := range cases {
		if w := postRestart(h, body, tc.headers); w.Code != tc.status {
			t.Fatalf("case %d: %d %s", i, w.Code, w.Body.String())
		}
	}
	if console.restart.Source != "ui" || console.restart.Reason != "frozen" {
		t.Fatalf("%+v", console.restart)
	}
	if w := postRestart(h, `{"workload":"vault"}`, json); w.Code != 400 || !strings.Contains(w.Body.String(), "cannot be restarted") {
		t.Fatal(w.Code, w.Body.String())
	}
	console.err = errors.New("secret patch failure")
	if w := postRestart(h, body, json); w.Code != 503 || strings.Contains(w.Body.String(), "secret") {
		t.Fatal(w.Code, w.Body.String())
	}
}

func TestSensitiveEndpointsRequireAllowedHost(t *testing.T) {
	h := NewHandler(nil, nil, &consoleStub{snapshot: stubSnapshot()}, "apps.corp.danielfrg.net")
	for _, tc := range []struct {
		method, url string
		status      int
	}{
		{"GET", "http://rebind.evil.example/api/logs?workload=frigate", 403},
		{"POST", "http://rebind.evil.example/api/restart", 403},
		{"GET", "http://localhost:5179/api/logs?workload=frigate", 200},
		{"GET", "http://apps.corp.danielfrg.net/api/logs?workload=frigate", 200},
		{"GET", "http://rebind.evil.example/api/overview", 200},
	} {
		r := httptest.NewRequest(tc.method, tc.url, strings.NewReader(`{"workload":"frigate"}`))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != tc.status {
			t.Fatalf("%s %s: %d", tc.method, tc.url, w.Code)
		}
	}
}
