package cluster

import (
	"math"
	"sort"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	pendingGrace  = 5 * time.Minute
	notReadyGrace = 2 * time.Minute
	recentWindow  = 24 * time.Hour
)

var blockingWaits = map[string]bool{
	"CrashLoopBackOff": true, "ImagePullBackOff": true, "ErrImagePull": true, "CreateContainerConfigError": true,
}

// Build joins raw cluster state into a snapshot. It is pure so the join and the
// problem rules can be tested without a cluster.
func Build(raw Raw, now time.Time) Snapshot {
	s := Snapshot{
		UpdatedAt:        now.UTC(),
		ArgoAvailable:    raw.ArgoErr == nil,
		MetricsAvailable: raw.MetricsErr == nil && raw.NodeUsage != nil,
		Apps:             []App{}, Workloads: []Workload{}, Pods: []Pod{}, Nodes: []Node{}, Events: []Event{},
	}
	links := map[string]Link{}
	for _, l := range raw.Links {
		links[l.ID] = Link{Name: l.Name, URL: l.URL, Aliases: append([]string{}, l.Aliases...), Description: l.Description, Group: l.Group, Icon: l.Icon}
	}
	owners := map[string]*App{} // "Kind/namespace/name" -> app
	for _, obj := range raw.Applications {
		app := buildApp(obj, links)
		s.Apps = append(s.Apps, app)
	}
	sort.Slice(s.Apps, func(i, j int) bool { return s.Apps[i].Name < s.Apps[j].Name })
	for i := range s.Apps {
		for _, key := range appResourceKeys(raw.Applications, s.Apps[i].Name) {
			owners[key] = &s.Apps[i]
		}
	}

	for _, d := range raw.Deployments {
		s.Workloads = append(s.Workloads, deploymentWorkload(d, owners))
	}
	for _, st := range raw.StatefulSets {
		s.Workloads = append(s.Workloads, statefulSetWorkload(st, owners))
	}
	sort.Slice(s.Workloads, func(i, j int) bool { return workloadKey(s.Workloads[i]) < workloadKey(s.Workloads[j]) })

	for _, p := range raw.Pods {
		pod := buildPod(p, now)
		if w := ownerWorkload(s.Workloads, p); w != nil {
			pod.Workload, pod.App = w.Name, w.App
			w.Pods = append(w.Pods, p.Name)
		}
		s.Pods = append(s.Pods, pod)
	}
	sort.Slice(s.Pods, func(i, j int) bool {
		return s.Pods[i].Namespace+"/"+s.Pods[i].Name < s.Pods[j].Namespace+"/"+s.Pods[j].Name
	})
	for i := range s.Workloads {
		sort.Strings(s.Workloads[i].Pods)
	}

	for _, n := range raw.Nodes {
		s.Nodes = append(s.Nodes, buildNode(n, raw.NodeUsage, s.MetricsAvailable))
	}
	sort.Slice(s.Nodes, func(i, j int) bool { return s.Nodes[i].Name < s.Nodes[j].Name })

	for _, e := range raw.Events {
		if e.Type != corev1.EventTypeWarning {
			continue
		}
		s.Events = append(s.Events, Event{
			Namespace: e.Namespace, Object: e.InvolvedObject.Kind + "/" + e.InvolvedObject.Name,
			Reason: e.Reason, Message: e.Message, Count: e.Count, LastSeen: eventTime(e),
		})
	}
	sort.SliceStable(s.Events, func(i, j int) bool { return s.Events[i].LastSeen.After(s.Events[j].LastSeen) })
	return s
}

// ownerWorkload finds the workload that owns a pod through its owner reference:
// a StatefulSet directly, or a Deployment through its "<name>-<pod-template-hash>"
// ReplicaSet. Selectors are only a fallback for pods without such an owner,
// because a broad selector can match a sibling workload's pods.
func ownerWorkload(workloads []Workload, p corev1.Pod) *Workload {
	for _, ref := range p.OwnerReferences {
		for i := range workloads {
			w := &workloads[i]
			if w.Namespace != p.Namespace {
				continue
			}
			switch {
			case ref.Kind == "StatefulSet" && w.Kind == "StatefulSet" && ref.Name == w.Name:
				return w
			case ref.Kind == "ReplicaSet" && w.Kind == "Deployment" && p.Labels["pod-template-hash"] != "" &&
				ref.Name == w.Name+"-"+p.Labels["pod-template-hash"]:
				return w
			}
		}
		if ref.Kind == "ReplicaSet" || ref.Kind == "StatefulSet" || ref.Kind == "DaemonSet" || ref.Kind == "Job" {
			return nil
		}
	}
	for i := range workloads {
		w := &workloads[i]
		if w.Namespace == p.Namespace && w.selector != nil && w.selector.Matches(labels.Set(p.Labels)) {
			return w
		}
	}
	return nil
}

func workloadKey(w Workload) string { return w.Namespace + "/" + w.Name + "/" + w.Kind }

func buildApp(obj unstructured.Unstructured, links map[string]Link) App {
	str := func(fields ...string) string { v, _, _ := unstructured.NestedString(obj.Object, fields...); return v }
	app := App{
		Name: obj.GetName(), Project: str("spec", "project"), Namespace: str("spec", "destination", "namespace"),
		Sync: str("status", "sync", "status"), Health: str("status", "health", "status"),
		OperationPhase: str("status", "operationState", "phase"), OperationMessage: str("status", "operationState", "message"),
		OperationFinishedAt: parseTime(str("status", "operationState", "finishedAt")),
		Revision:            str("status", "sync", "revision"),
		Images:              []string{}, Links: []Link{}, Workloads: []string{},
		RestartOptOut: isOptOut(obj.GetAnnotations()),
		Source:        appSource(obj),
	}
	if history, _, _ := unstructured.NestedSlice(obj.Object, "status", "history"); len(history) > 0 {
		if last, ok := history[len(history)-1].(map[string]any); ok {
			if at, ok := last["deployedAt"].(string); ok {
				app.DeployedAt = parseTime(at)
			}
			if rev, ok := last["revision"].(string); ok && rev != "" {
				app.Revision = rev
			}
		}
	}
	images, _, _ := unstructured.NestedStringSlice(obj.Object, "status", "summary", "images")
	app.Images = append(app.Images, images...)
	resources, _, _ := unstructured.NestedSlice(obj.Object, "status", "resources")
	for _, item := range resources {
		r, ok := item.(map[string]any)
		if !ok {
			continue
		}
		kind, _ := r["kind"].(string)
		namespace, _ := r["namespace"].(string)
		name, _ := r["name"].(string)
		switch kind {
		case "Deployment", "StatefulSet":
			app.Workloads = append(app.Workloads, namespace+"/"+name)
		case "HTTPRoute":
			if link, ok := links[namespace+"/"+name]; ok {
				app.Links = append(app.Links, link)
			}
		}
	}
	sort.Strings(app.Workloads)
	return app
}

func appResourceKeys(applications []unstructured.Unstructured, name string) []string {
	keys := []string{}
	for _, obj := range applications {
		if obj.GetName() != name {
			continue
		}
		resources, _, _ := unstructured.NestedSlice(obj.Object, "status", "resources")
		for _, item := range resources {
			if r, ok := item.(map[string]any); ok {
				kind, _ := r["kind"].(string)
				namespace, _ := r["namespace"].(string)
				n, _ := r["name"].(string)
				keys = append(keys, kind+"/"+namespace+"/"+n)
			}
		}
	}
	return keys
}

func appSource(obj unstructured.Unstructured) string {
	sources := []map[string]any{}
	if src, ok, _ := unstructured.NestedMap(obj.Object, "spec", "source"); ok {
		sources = append(sources, src)
	}
	if list, ok, _ := unstructured.NestedSlice(obj.Object, "spec", "sources"); ok {
		for _, item := range list {
			if m, ok := item.(map[string]any); ok {
				sources = append(sources, m)
			}
		}
	}
	for _, src := range sources {
		if chart, _ := src["chart"].(string); chart != "" {
			version, _ := src["targetRevision"].(string)
			return "chart " + chart + "@" + version
		}
	}
	for _, src := range sources {
		if path, _ := src["path"].(string); path != "" {
			repo, _ := src["repoURL"].(string)
			return repo + " " + path
		}
	}
	return ""
}

func isOptOut(annotations map[string]string) bool {
	return strings.EqualFold(strings.TrimSpace(annotations[optOutAnnotation]), "false")
}

func restartability(w *Workload, owner *App, annotations map[string]string) {
	switch {
	case owner == nil:
		w.NotRestartableReason = "not part of an Argo CD app"
	case owner.RestartOptOut || isOptOut(annotations):
		w.NotRestartableReason = "opted out with " + optOutAnnotation
	case w.Strategy == string(appsv1.OnDeleteStatefulSetStrategyType):
		w.NotRestartableReason = "uses OnDelete updates; a restart would not roll its pods"
	default:
		w.Restartable = true
	}
}

func deploymentWorkload(d appsv1.Deployment, owners map[string]*App) Workload {
	desired := int32(1)
	if d.Spec.Replicas != nil {
		desired = *d.Spec.Replicas
	}
	strategy := string(d.Spec.Strategy.Type)
	if strategy == "" {
		strategy = string(appsv1.RollingUpdateDeploymentStrategyType)
	}
	w := Workload{
		Kind: "Deployment", Namespace: d.Namespace, Name: d.Name, Strategy: strategy,
		Desired: desired, Ready: d.Status.ReadyReplicas, Updated: d.Status.UpdatedReplicas, Pods: []string{},
		RestartedAt: parseTime(d.Spec.Template.Annotations[restartAnnotation]),
		inProgress:  d.Status.ObservedGeneration < d.Generation || d.Status.UpdatedReplicas < desired,
	}
	w.selector = selector(d.Spec.Selector)
	stuck := false
	for _, c := range d.Status.Conditions {
		if c.Type == appsv1.DeploymentProgressing && c.Status == corev1.ConditionFalse && c.Reason == "ProgressDeadlineExceeded" {
			stuck = true
		}
	}
	w.Rollout = rollout(stuck, w.inProgress)
	owner := owners["Deployment/"+d.Namespace+"/"+d.Name]
	if owner != nil {
		w.App = owner.Name
	}
	restartability(&w, owner, d.Annotations)
	return w
}

func statefulSetWorkload(st appsv1.StatefulSet, owners map[string]*App) Workload {
	desired := int32(1)
	if st.Spec.Replicas != nil {
		desired = *st.Spec.Replicas
	}
	strategy := string(st.Spec.UpdateStrategy.Type)
	if strategy == "" {
		strategy = string(appsv1.RollingUpdateStatefulSetStrategyType)
	}
	inProgress := st.Status.ObservedGeneration < st.Generation
	if strategy != string(appsv1.OnDeleteStatefulSetStrategyType) && st.Status.UpdatedReplicas < desired {
		inProgress = true
	}
	w := Workload{
		Kind: "StatefulSet", Namespace: st.Namespace, Name: st.Name, Strategy: strategy,
		Desired: desired, Ready: st.Status.ReadyReplicas, Updated: st.Status.UpdatedReplicas, Pods: []string{},
		RestartedAt: parseTime(st.Spec.Template.Annotations[restartAnnotation]), inProgress: inProgress,
	}
	w.selector = selector(st.Spec.Selector)
	w.Rollout = rollout(false, inProgress)
	owner := owners["StatefulSet/"+st.Namespace+"/"+st.Name]
	if owner != nil {
		w.App = owner.Name
	}
	restartability(&w, owner, st.Annotations)
	return w
}

func rollout(stuck, inProgress bool) string {
	switch {
	case stuck:
		return "stuck"
	case inProgress:
		return "progressing"
	default:
		return "complete"
	}
}

func selector(s *metav1.LabelSelector) labels.Selector {
	if s == nil {
		return nil
	}
	sel, err := metav1.LabelSelectorAsSelector(s)
	if err != nil || sel.Empty() {
		return nil
	}
	return sel
}

func buildPod(p corev1.Pod, now time.Time) Pod {
	pod := Pod{Namespace: p.Namespace, Name: p.Name, Phase: string(p.Status.Phase), Node: p.Spec.NodeName, Images: []string{}, Containers: []string{}}
	for _, c := range p.Spec.Containers {
		pod.Containers = append(pod.Containers, c.Name)
	}
	if p.Status.StartTime != nil {
		t := p.Status.StartTime.UTC()
		pod.StartedAt = &t
	}
	var readySince time.Time
	for _, c := range p.Status.Conditions {
		if c.Type == corev1.PodReady {
			pod.Ready = c.Status == corev1.ConditionTrue
			readySince = c.LastTransitionTime.Time
		}
	}
	for _, c := range p.Status.ContainerStatuses {
		if c.LastTerminationState.Terminated != nil {
			pod.terminated = append(pod.terminated, c.Name)
		}
		pod.Restarts += c.RestartCount
		pod.Images = append(pod.Images, c.Image)
		if c.State.Waiting != nil && pod.Waiting == "" {
			pod.Waiting = c.State.Waiting.Reason
		}
		if t := c.LastTerminationState.Terminated; t != nil {
			if pod.LastTermination == nil || t.FinishedAt.After(pod.LastTermination.FinishedAt) {
				pod.LastTermination = &Termination{Reason: t.Reason, ExitCode: t.ExitCode, FinishedAt: t.FinishedAt.UTC()}
			}
		}
	}
	if len(pod.Images) == 0 {
		for _, c := range p.Spec.Containers {
			pod.Images = append(pod.Images, c.Image)
		}
	}
	age := now.Sub(p.CreationTimestamp.Time)
	switch {
	case blockingWaits[pod.Waiting]:
		pod.Problem = pod.Waiting
	case p.Status.Phase == corev1.PodPending && age > pendingGrace:
		pod.Problem = "Pending"
	case p.Status.Phase == corev1.PodFailed && age < recentWindow:
		pod.Problem = "Failed"
	case p.Status.Phase == corev1.PodRunning && !pod.Ready && !readySince.IsZero() && now.Sub(readySince) > notReadyGrace:
		pod.Problem = "NotReady"
	case pod.LastTermination != nil && now.Sub(pod.LastTermination.FinishedAt) < recentWindow:
		if pod.LastTermination.Reason == "OOMKilled" {
			pod.Problem = "OOMKilled"
		} else {
			pod.Problem = "Restarted"
		}
	}
	return pod
}

func buildNode(n corev1.Node, usage map[string]NodeUsage, metrics bool) Node {
	node := Node{
		Name: n.Name, KubeletVersion: n.Status.NodeInfo.KubeletVersion, KernelVersion: n.Status.NodeInfo.KernelVersion,
		Role: n.Labels["homelab.danielfrg.net/role"], Pressure: []string{},
	}
	if _, ok := n.Labels["node-role.kubernetes.io/control-plane"]; ok && node.Role == "" {
		node.Role = "control-plane"
	}
	for _, c := range n.Status.Conditions {
		switch {
		case c.Type == corev1.NodeReady:
			node.Ready = c.Status == corev1.ConditionTrue
		case c.Status == corev1.ConditionTrue && strings.HasSuffix(string(c.Type), "Pressure"):
			node.Pressure = append(node.Pressure, string(c.Type))
		}
	}
	if u, ok := usage[n.Name]; metrics && ok {
		if cpu := n.Status.Allocatable.Cpu().MilliValue(); cpu > 0 {
			node.CPUPercent = percent(u.CPUMilli, cpu)
		}
		if mem := n.Status.Allocatable.Memory().Value(); mem > 0 {
			node.MemoryPercent = percent(u.MemoryBytes, mem)
		}
	}
	return node
}

func percent(used, total int64) *float64 {
	v := math.Round(float64(used)/float64(total)*1000) / 10
	return &v
}

func eventTime(e corev1.Event) time.Time {
	switch {
	case !e.LastTimestamp.IsZero():
		return e.LastTimestamp.UTC()
	case !e.EventTime.IsZero():
		return e.EventTime.UTC()
	default:
		return e.CreationTimestamp.UTC()
	}
}

func parseTime(value string) *time.Time {
	if value == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil
	}
	t = t.UTC()
	return &t
}
