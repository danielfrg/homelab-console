package cluster

import (
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"
)

const maxProblems = 50

type AppSummary struct {
	Name       string     `json:"name"`
	Sync       string     `json:"sync"`
	Health     string     `json:"health"`
	Ready      string     `json:"ready" doc:"Ready workloads over total, like 4/4"`
	Link       string     `json:"link"`
	DeployedAt *time.Time `json:"deployed_at"`
}

type Problem struct {
	Kind      string `json:"kind" doc:"node, workload, or pod"`
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name"`
	App       string `json:"app,omitempty"`
	Reason    string `json:"reason"`
}

type RecentRestart struct {
	App         string    `json:"app"`
	Namespace   string    `json:"namespace"`
	Name        string    `json:"name"`
	RestartedAt time.Time `json:"restarted_at"`
}

type Overview struct {
	UpdatedAt        time.Time       `json:"updated_at"`
	Stale            bool            `json:"stale"`
	ArgoAvailable    bool            `json:"argo_available"`
	MetricsAvailable bool            `json:"metrics_available"`
	Nodes            []Node          `json:"nodes" nullable:"false"`
	AppTotals        map[string]int  `json:"app_totals"`
	NeedsAttention   []AppSummary    `json:"needs_attention" nullable:"false"`
	Problems         []Problem       `json:"problems" nullable:"false"`
	ProblemTotal     int             `json:"problem_total"`
	RecentRestarts   []RecentRestart `json:"recent_restarts" nullable:"false"`
}

// Summaries returns one compact line per app, in name order.
func (s Snapshot) Summaries() []AppSummary {
	byKey := map[string]Workload{}
	for _, w := range s.Workloads {
		byKey[w.Namespace+"/"+w.Name] = w
	}
	out := make([]AppSummary, 0, len(s.Apps))
	for _, a := range s.Apps {
		ready := 0
		for _, key := range a.Workloads {
			if w, ok := byKey[key]; ok && w.Ready >= w.Desired {
				ready++
			}
		}
		summary := AppSummary{Name: a.Name, Sync: a.Sync, Health: a.Health, DeployedAt: a.DeployedAt,
			Ready: fmt.Sprintf("%d/%d", ready, len(a.Workloads))}
		if len(a.Links) > 0 {
			summary.Link = a.Links[0].URL
		}
		out = append(out, summary)
	}
	return out
}

// Overview answers "is everything OK?": node health, app totals, and the
// problems worth a look, nodes first, then workloads, then pods.
func (s Snapshot) Overview(now time.Time) Overview {
	o := Overview{
		UpdatedAt: s.UpdatedAt, Stale: s.Stale, ArgoAvailable: s.ArgoAvailable, MetricsAvailable: s.MetricsAvailable,
		Nodes: s.Nodes, AppTotals: map[string]int{}, NeedsAttention: []AppSummary{}, Problems: []Problem{}, RecentRestarts: []RecentRestart{},
	}
	for _, summary := range s.Summaries() {
		o.AppTotals[snake(summary.Sync)]++
		o.AppTotals[snake(summary.Health)]++
		if summary.Sync != "Synced" || summary.Health != "Healthy" {
			o.NeedsAttention = append(o.NeedsAttention, summary)
		}
	}
	problems := []Problem{}
	for _, n := range s.Nodes {
		if !n.Ready {
			problems = append(problems, Problem{Kind: "node", Name: n.Name, Reason: "NotReady"})
		}
		for _, p := range n.Pressure {
			problems = append(problems, Problem{Kind: "node", Name: n.Name, Reason: p})
		}
	}
	for _, w := range s.Workloads {
		switch {
		case w.Rollout == "stuck":
			problems = append(problems, Problem{Kind: "workload", Namespace: w.Namespace, Name: w.Name, App: w.App, Reason: "rollout stuck"})
		case w.Desired > 0 && w.Ready < w.Desired:
			problems = append(problems, Problem{Kind: "workload", Namespace: w.Namespace, Name: w.Name, App: w.App,
				Reason: fmt.Sprintf("%d/%d ready", w.Ready, w.Desired)})
		}
		if w.RestartedAt != nil && now.Sub(*w.RestartedAt) < recentWindow {
			o.RecentRestarts = append(o.RecentRestarts, RecentRestart{App: w.App, Namespace: w.Namespace, Name: w.Name, RestartedAt: *w.RestartedAt})
		}
	}
	for _, p := range s.Pods {
		if p.Problem != "" {
			problems = append(problems, Problem{Kind: "pod", Namespace: p.Namespace, Name: p.Name, App: p.App, Reason: p.Problem})
		}
	}
	sort.SliceStable(o.RecentRestarts, func(i, j int) bool { return o.RecentRestarts[i].RestartedAt.After(o.RecentRestarts[j].RestartedAt) })
	o.ProblemTotal = len(problems)
	o.Problems = problems[:min(len(problems), maxProblems)]
	return o
}

func snake(value string) string {
	var b strings.Builder
	for i, r := range value {
		if unicode.IsUpper(r) && i > 0 {
			b.WriteByte('_')
		}
		b.WriteRune(unicode.ToLower(r))
	}
	if b.Len() == 0 {
		return "unknown"
	}
	return b.String()
}
