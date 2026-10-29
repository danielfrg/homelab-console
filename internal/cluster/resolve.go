package cluster

import (
	"errors"
	"sort"
	"strings"
)

var ErrNotFound = errors.New("not found")

// AmbiguousError lists every workload a bare name matched, as "namespace/name (app)".
type AmbiguousError struct{ Candidates []string }

func (e *AmbiguousError) Error() string {
	return "matches several workloads: " + strings.Join(e.Candidates, ", ")
}

// FindApp resolves an Argo app by name, or the app owning a uniquely named
// workload. The second result names that workload, or is empty.
func (s Snapshot) FindApp(name string) (App, string, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	for _, a := range s.Apps {
		if strings.ToLower(a.Name) == name {
			return a, "", nil
		}
	}
	w, err := s.FindWorkload(name)
	if err != nil {
		return App{}, "", err
	}
	for _, a := range s.Apps {
		if a.Name == w.App {
			return a, w.Name, nil
		}
	}
	return App{}, "", ErrNotFound
}

// FindWorkload resolves "name" or "namespace/name", case-insensitively.
func (s Snapshot) FindWorkload(ref string) (Workload, error) {
	ref = strings.ToLower(strings.TrimSpace(ref))
	namespace, name, qualified := strings.Cut(ref, "/")
	if !qualified {
		name, namespace = namespace, ""
	}
	matches := []Workload{}
	for _, w := range s.Workloads {
		if strings.ToLower(w.Name) == name && (namespace == "" || strings.ToLower(w.Namespace) == namespace) {
			matches = append(matches, w)
		}
	}
	switch len(matches) {
	case 0:
		return Workload{}, ErrNotFound
	case 1:
		return matches[0], nil
	}
	candidates := []string{}
	for _, w := range matches {
		app := w.App
		if app == "" {
			app = "no app"
		}
		candidates = append(candidates, w.Namespace+"/"+w.Name+" ("+app+")")
	}
	sort.Strings(candidates)
	return Workload{}, &AmbiguousError{Candidates: candidates}
}
