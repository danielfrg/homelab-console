package mcp

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/danielfrg/homelab-console/internal/cluster"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

var testTime = time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)

type stubConsole struct {
	snapshot cluster.Snapshot
	err      error
	enabled  bool
	restart  cluster.RestartRequest
	logs     cluster.LogsRequest
}

func (c *stubConsole) Snapshot(_ context.Context, refresh bool) (cluster.Snapshot, error) {
	if refresh {
		panic("tools must not force a refresh")
	}
	return c.snapshot, c.err
}
func (c *stubConsole) Logs(_ context.Context, req cluster.LogsRequest) (cluster.LogsResult, error) {
	c.logs = req
	if c.err != nil {
		return cluster.LogsResult{}, c.err
	}
	return cluster.LogsResult{Pod: "frigate-abc", Container: "frigate", OtherPods: []string{}, Text: "ok\n"}, nil
}
func (c *stubConsole) Restart(_ context.Context, req cluster.RestartRequest) (cluster.RestartResult, error) {
	c.restart = req
	if c.err != nil {
		return cluster.RestartResult{}, c.err
	}
	return cluster.RestartResult{Restarted: []cluster.RestartedWorkload{{Kind: "Deployment", Namespace: "default", Name: req.Workload, RestartedAt: testTime}}, Message: "Rollout started; check get_app for progress."}, nil
}
func (c *stubConsole) RestartEnabled() bool { return c.enabled }

func testSnapshot() cluster.Snapshot {
	return cluster.Snapshot{
		UpdatedAt: testTime, ArgoAvailable: true, MetricsAvailable: true,
		Apps: []cluster.App{
			{Name: "home-assistant", Sync: "Synced", Health: "Healthy", Images: []string{}, Workloads: []string{"default/frigate", "default/home-assistant"},
				Links: []cluster.Link{{Name: "Home Assistant", URL: "https://ha.corp.danielfrg.net", Aliases: []string{"home.corp.danielfrg.net"}}}},
			{Name: "sofigram", Sync: "Synced", Health: "Degraded", Images: []string{}, Links: []cluster.Link{}, Workloads: []string{"default/sofigram"}},
		},
		Workloads: []cluster.Workload{
			{App: "home-assistant", Kind: "Deployment", Namespace: "default", Name: "frigate", Desired: 1, Ready: 1, Rollout: "complete", Restartable: true, Pods: []string{"frigate-abc"}},
			{App: "home-assistant", Kind: "Deployment", Namespace: "default", Name: "home-assistant", Desired: 1, Ready: 1, Rollout: "complete", Restartable: true, Pods: []string{"home-assistant-abc"}},
			{App: "sofigram", Kind: "Deployment", Namespace: "default", Name: "sofigram", Desired: 1, Ready: 0, Rollout: "complete", Restartable: true, Pods: []string{"sofigram-abc"}},
		},
		Pods: []cluster.Pod{
			{Namespace: "default", Name: "frigate-abc", App: "home-assistant", Workload: "frigate", Phase: "Running", Node: "srv-nuc-02", Ready: true, Images: []string{}, Containers: []string{"frigate"}},
			{Namespace: "default", Name: "home-assistant-abc", App: "home-assistant", Workload: "home-assistant", Phase: "Running", Node: "srv-nuc-01", Ready: true, Images: []string{}, Containers: []string{"home-assistant"}},
			{Namespace: "default", Name: "sofigram-abc", App: "sofigram", Workload: "sofigram", Phase: "Running", Node: "srv-nuc-02", Problem: "CrashLoopBackOff", Images: []string{}, Containers: []string{"sofigram"}},
		},
		Nodes:  []cluster.Node{{Name: "srv-nuc-01", Ready: true, Pressure: []string{}}},
		Events: []cluster.Event{{Namespace: "default", Object: "Pod/sofigram-abc", Reason: "BackOff", Message: "Back-off", LastSeen: testTime}},
	}
}

func TestConsoleTools(t *testing.T) {
	console := &stubConsole{snapshot: testSnapshot(), enabled: true}
	s := session(t, NewHandler(console, nil), 6)
	overview := call(t, s, "get_overview", map[string]any{}, false)
	if problems := overview["problems"].([]any); len(problems) != 2 {
		t.Fatalf("problems %v", problems)
	}
	apps := call(t, s, "list_apps", map[string]any{"query": "FRIGATE"}, false)
	if apps["total"] != float64(1) || apps["apps"].([]any)[0].(map[string]any)["name"] != "home-assistant" {
		t.Fatal(apps)
	}
	if v := call(t, s, "list_apps", map[string]any{"query": "home.corp"}, false); v["total"] != float64(1) {
		t.Fatal(v)
	}
	problems := call(t, s, "list_apps", map[string]any{"status": "problems"}, false)
	if problems["total"] != float64(1) || problems["apps"].([]any)[0].(map[string]any)["name"] != "sofigram" {
		t.Fatal(problems)
	}
	call(t, s, "list_apps", map[string]any{"status": "weird"}, true)
	app := call(t, s, "get_app", map[string]any{"name": "frigate"}, false)
	if app["matched_workload"] != "frigate" || len(app["workloads"].([]any)) != 2 {
		t.Fatal(app)
	}
	sofigram := call(t, s, "get_app", map[string]any{"name": "sofigram"}, false)
	if len(sofigram["events"].([]any)) != 1 || len(sofigram["workloads"].([]any)[0].(map[string]any)["pods"].([]any)) != 1 {
		t.Fatal(sofigram)
	}
	call(t, s, "get_app", map[string]any{"name": "nope"}, true)
	pods := call(t, s, "list_pods", map[string]any{"problems_only": true}, false)
	if pods["total"] != float64(1) {
		t.Fatal(pods)
	}
	if v := call(t, s, "list_pods", map[string]any{"app": "home-assistant", "node": "srv-nuc-02"}, false); v["total"] != float64(1) {
		t.Fatal(v)
	}
	page := call(t, s, "list_pods", map[string]any{"limit": 1}, false)
	if page["next_offset"] != float64(1) || page["total"] != float64(3) {
		t.Fatal(page)
	}
	for _, limit := range []int{0, 101} {
		call(t, s, "list_pods", map[string]any{"limit": limit}, true)
	}
	logs := call(t, s, "get_logs", map[string]any{"workload": "frigate", "previous": true, "lines": 50}, false)
	if logs["text"] != "ok\n" || !console.logs.Previous || console.logs.Lines != 50 {
		t.Fatal(logs, console.logs)
	}
	restarted := call(t, s, "restart", map[string]any{"workload": "frigate", "reason": "frozen"}, false)
	if console.restart.Source != "mcp" || console.restart.Reason != "frozen" || len(restarted["restarted"].([]any)) != 1 {
		t.Fatal(restarted, console.restart)
	}
}

func TestRestartToolDisabled(t *testing.T) {
	s := session(t, NewHandler(&stubConsole{snapshot: testSnapshot()}, nil), 5)
	v, err := s.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range v.Tools {
		if tool.Name == "restart" {
			t.Fatal("restart must not be registered")
		}
	}
}

func TestToolErrorsSanitized(t *testing.T) {
	console := &stubConsole{err: errors.New("secret token leaked"), enabled: true}
	s := session(t, NewHandler(console, nil), 6)
	for _, tool := range []string{"get_overview", "list_apps", "get_logs", "restart"} {
		args := map[string]any{}
		if tool == "get_logs" || tool == "restart" {
			args["workload"] = "frigate"
		}
		if text := errorText(t, s, tool, args); strings.Contains(text, "secret") {
			t.Fatalf("%s leaked: %s", tool, text)
		}
	}
	console.err = &cluster.UserError{Message: "restart is disabled"}
	if text := errorText(t, s, "restart", map[string]any{"workload": "frigate"}); !strings.Contains(text, "restart is disabled") {
		t.Fatal(text)
	}
}

func errorText(t *testing.T, s *sdk.ClientSession, name string, args any) string {
	t.Helper()
	v, err := s.CallTool(context.Background(), &sdk.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatal(err)
	}
	if !v.IsError {
		t.Fatalf("%s: expected error", name)
	}
	text := ""
	for _, c := range v.Content {
		if tc, ok := c.(*sdk.TextContent); ok {
			text += tc.Text
		}
	}
	return text
}
