package cluster

import (
	"fmt"
	"testing"
	"time"
)

func TestOverviewTotalsAndAttention(t *testing.T) {
	o := Build(fixture(), testNow).Overview(testNow)
	want := map[string]int{"synced": 3, "out_of_sync": 1, "healthy": 3, "degraded": 1}
	for k, v := range want {
		if o.AppTotals[k] != v {
			t.Fatalf("totals %v", o.AppTotals)
		}
	}
	if len(o.NeedsAttention) != 2 || o.NeedsAttention[0].Name != "plex" || o.NeedsAttention[1].Name != "sofigram" {
		t.Fatalf("attention %+v", o.NeedsAttention)
	}
	if o.NeedsAttention[1].Ready != "0/1" {
		t.Fatalf("ready %q", o.NeedsAttention[1].Ready)
	}
	if len(o.Nodes) != 2 || !o.ArgoAvailable || !o.MetricsAvailable {
		t.Fatalf("overview %+v", o)
	}
}

func TestOverviewProblems(t *testing.T) {
	raw := fixture()
	raw.Nodes = append(raw.Nodes, node("srv-rpi-01", false))
	o := Build(raw, testNow).Overview(testNow)
	has := func(kind, name, reason string) bool {
		for _, p := range o.Problems {
			if p.Kind == kind && p.Name == name && p.Reason == reason {
				return true
			}
		}
		return false
	}
	if !has("pod", "sofigram-abc", "CrashLoopBackOff") || !has("workload", "sofigram", "0/1 ready") || !has("node", "srv-rpi-01", "NotReady") {
		t.Fatalf("problems %+v", o.Problems)
	}
	if p := o.Problems[0]; p.Kind != "node" {
		t.Fatalf("nodes first, got %+v", p)
	}
	for i := 0; i < 80; i++ {
		raw.Pods = append(raw.Pods, crashing(pod("default", fmt.Sprintf("crash-%02d", i), "none", testNow.Add(-time.Hour))))
	}
	if o = Build(raw, testNow).Overview(testNow); len(o.Problems) != 50 || o.ProblemTotal < 83 {
		t.Fatalf("cap %d total %d", len(o.Problems), o.ProblemTotal)
	}
}

func TestOverviewRecentRestarts(t *testing.T) {
	raw := fixture()
	raw.Deployments[0].Spec.Template.Annotations = map[string]string{restartAnnotation: testNow.Add(-3 * time.Hour).Format(time.RFC3339)}
	raw.Deployments[1].Spec.Template.Annotations = map[string]string{restartAnnotation: testNow.Add(-25 * time.Hour).Format(time.RFC3339)}
	o := Build(raw, testNow).Overview(testNow)
	if len(o.RecentRestarts) != 1 || o.RecentRestarts[0].Name != "frigate" || o.RecentRestarts[0].App != "home-assistant" {
		t.Fatalf("restarts %+v", o.RecentRestarts)
	}
}
