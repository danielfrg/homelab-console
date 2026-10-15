package apps

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/rest"
)

func route(namespace, name string, hosts []string, annotations map[string]string) unstructured.Unstructured {
	hostValues := make([]any, len(hosts))
	for i, host := range hosts {
		hostValues[i] = host
	}
	annotationValues := map[string]any{}
	for key, value := range annotations {
		annotationValues[key] = value
	}
	return unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "gateway.networking.k8s.io/v1", "kind": "HTTPRoute",
		"metadata": map[string]any{"namespace": namespace, "name": name, "annotations": annotationValues,
			"labels": map[string]any{"app.kubernetes.io/name": "restic"}},
		"spec": map[string]any{"hostnames": hostValues},
	}}
}

func TestNormalizeRoutes(t *testing.T) {
	routes := &unstructured.UnstructuredList{Items: []unstructured.Unstructured{
		route("apps", "files-http", []string{"*.example.test", "files.example.test", "files.other.test", "files.example.test"}, map[string]string{
			"homelab-launcher/name":         "File Browser",
			"homelab-launcher/description":  "Shared files",
			"homelab-launcher/group":        "Tools",
			"homelab-launcher/icon":         "mdi-folder",
			"unrelated.example.test/secret": "never expose this",
		}),
		route("apps", "calendar-http", []string{"calendar.example.org"}, nil),
		route("apps", "launcher-http", []string{"launcher.example.test"}, nil),
		route("apps", "hidden-http", []string{"hidden.example.test"}, map[string]string{"homelab-launcher/enabled": " FaLsE "}),
		route("apps", "invalid-http", []string{"", "*.example.test", "https://invalid.example.test", "bad_host.example.test", "-invalid.example.test"}, nil),
		route("apps", "hostless-http", nil, nil),
	}}
	want := []App{
		{ID: "apps/calendar-http", Name: "Calendar", URL: "https://calendar.example.org", Aliases: []string{}, Group: "Apps"},
		{ID: "apps/files-http", Name: "File Browser", URL: "https://files.example.test", Aliases: []string{"files.other.test"}, Description: "Shared files", Group: "Tools", Icon: "mdi-folder"},
		{ID: "apps/launcher-http", Name: "Launcher", URL: "https://launcher.example.test", Aliases: []string{}, Group: "Apps"},
	}
	if got := normalizeRoutes(routes); !reflect.DeepEqual(got, want) {
		t.Fatalf("apps = %#v, want %#v", got, want)
	}
}

func TestKubernetesSourceListsOnlyRoutes(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.RequestURI())
		if r.Method != "GET" || r.URL.Path != "/apis/gateway.networking.k8s.io/v1/httproutes" {
			t.Errorf("unexpected request %s", r.URL)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"apiVersion": "gateway.networking.k8s.io/v1", "kind": "HTTPRouteList", "metadata": map[string]any{},
			"items": []any{route("default", "calendar-http", []string{"calendar.example.test"}, nil).Object},
		})
	}))
	defer server.Close()
	source, err := NewKubernetesSource(&rest.Config{Host: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	got, err := source.List(context.Background())
	if err != nil || len(got) != 1 || got[0].ID != "default/calendar-http" || !reflect.DeepEqual(requests, []string{"GET /apis/gateway.networking.k8s.io/v1/httproutes?timeout=5s"}) {
		t.Fatalf("apps = %+v, error %v, requests %v", got, err, requests)
	}
}

func TestKubernetesSourceRejectsForbiddenAndHonorsDeadline(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("slow") != "" {
			<-r.Context().Done()
			return
		}
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"kind":"Status","apiVersion":"v1","status":"Failure","reason":"Forbidden","message":"forbidden","code":403}`))
	}))
	defer server.Close()
	source, err := NewKubernetesSource(&rest.Config{Host: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := source.List(context.Background()); err == nil {
		t.Fatal("forbidden list succeeded")
	}
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer slow.Close()
	source, err = NewKubernetesSource(&rest.Config{Host: slow.URL})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	if _, err := source.List(ctx); err == nil {
		t.Fatal("deadline list succeeded")
	}
	if time.Since(start) > time.Second {
		t.Fatal("caller deadline was not respected")
	}
}

func TestKubernetesConfigUsesKubeconfigCurrentContext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config")
	contents := `apiVersion: v1
kind: Config
clusters:
  - name: ignored
    cluster:
      server: https://ignored.example.com
  - name: homelab
    cluster:
      server: https://cluster.example.com
contexts:
  - name: local-test
    context:
      cluster: homelab
      user: developer
users:
  - name: developer
    user:
      token: test-only-token
current-context: local-test
`
	if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KUBERNETES_SERVICE_HOST", "")
	t.Setenv("KUBERNETES_SERVICE_PORT", "")
	t.Setenv("KUBECONFIG", path)
	config, err := KubernetesConfig()
	if err != nil {
		t.Fatal(err)
	}
	if config.Host != "https://cluster.example.com" || config.BearerToken != "test-only-token" || config.Insecure {
		t.Fatal("current context or secure local credentials were not loaded correctly")
	}
}
