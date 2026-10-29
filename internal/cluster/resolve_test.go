package cluster

import (
	"errors"
	"strings"
	"testing"
)

func TestFindAppLenient(t *testing.T) {
	s := Build(fixture(), testNow)
	app, workload, err := s.FindApp("Frigate")
	if err != nil || app.Name != "home-assistant" || workload != "frigate" {
		t.Fatalf("%v %q %v", app.Name, workload, err)
	}
	app, workload, err = s.FindApp(" HOME-ASSISTANT ")
	if err != nil || app.Name != "home-assistant" || workload != "" {
		t.Fatalf("%v %q %v", app.Name, workload, err)
	}
	if _, _, err = s.FindApp("coredns"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("orphan workload should not resolve to an app: %v", err)
	}
	if _, _, err = s.FindApp("nope"); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
}

func TestFindWorkload(t *testing.T) {
	raw := fixture()
	raw.Deployments = append(raw.Deployments, deployment("default", "x", 1, 1, 1), deployment("other", "x", 1, 1, 1))
	raw.Applications = append(raw.Applications, application("app-a", "Synced", "Healthy", nil, resourceRef{"Deployment", "default", "x"}))
	s := Build(raw, testNow)
	if w, err := s.FindWorkload("frigate"); err != nil || w.Namespace != "default" {
		t.Fatal(w, err)
	}
	if w, err := s.FindWorkload("default/Frigate"); err != nil || w.Name != "frigate" {
		t.Fatal(w, err)
	}
	if w, err := s.FindWorkload("other/x"); err != nil || w.Namespace != "other" {
		t.Fatal(w, err)
	}
	_, err := s.FindWorkload("x")
	var ambiguous *AmbiguousError
	if !errors.As(err, &ambiguous) || !strings.Contains(err.Error(), "default/x (app-a)") || !strings.Contains(err.Error(), "other/x (no app)") {
		t.Fatalf("%v", err)
	}
	if _, err = s.FindWorkload("nope"); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
}
