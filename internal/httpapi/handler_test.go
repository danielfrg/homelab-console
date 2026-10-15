package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/danielfrg/homelab-launcher/internal/apps"
	"mime"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

func TestHello(t *testing.T) {
	response := httptest.NewRecorder()
	NewHandler(nil, nil).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/hello", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	mediaType, _, err := mime.ParseMediaType(response.Header().Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		t.Fatalf("content type = %q, error = %v", mediaType, err)
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body) != 1 || body["message"] != "Hello from Homelab" {
		t.Fatalf("body = %#v, want only the Homelab greeting", body)
	}
}

func TestRouting(t *testing.T) {
	handler := NewHandler(nil, nil)
	for _, tc := range []struct {
		method string
		path   string
		status int
	}{
		{http.MethodPost, "/api/hello", http.StatusMethodNotAllowed},
		{http.MethodGet, "/api/missing", http.StatusNotFound},
		{http.MethodHead, "/api/hello", http.StatusOK},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(tc.method, tc.path, nil))
			if response.Code != tc.status {
				t.Fatalf("status = %d, want %d", response.Code, tc.status)
			}
		})
	}
}

func TestOpenAPI(t *testing.T) {
	response := httptest.NewRecorder()
	NewHandler(nil, nil).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/openapi.json", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	var spec map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &spec); err != nil {
		t.Fatal(err)
	}
	version, _ := spec["openapi"].(string)
	if !strings.HasPrefix(version, "3.1.") {
		t.Fatalf("OpenAPI version = %q, want 3.1", version)
	}
	info := object(t, spec["info"])
	if info["title"] != "Homelab API" || info["version"] != "0.1.0" {
		t.Fatalf("info = %#v", info)
	}
	paths := object(t, spec["paths"])
	operation := object(t, object(t, paths["/api/hello"])["get"])
	if operation["operationId"] != "get-hello" {
		t.Fatalf("operation ID = %v", operation["operationId"])
	}
	responses := object(t, operation["responses"])
	content := object(t, object(t, responses["200"])["content"])
	schema := object(t, object(t, content["application/json"])["schema"])
	if ref, ok := schema["$ref"].(string); ok {
		const prefix = "#/components/schemas/"
		if !strings.HasPrefix(ref, prefix) {
			t.Fatalf("unsupported schema reference %q", ref)
		}
		schemas := object(t, object(t, spec["components"])["schemas"])
		schema = object(t, schemas[strings.TrimPrefix(ref, prefix)])
	}
	if schema["type"] != "object" {
		t.Fatalf("response schema type = %v", schema["type"])
	}
	properties := object(t, schema["properties"])
	if object(t, properties["message"])["type"] != "string" {
		t.Fatalf("message schema = %#v", properties["message"])
	}
	required, ok := schema["required"].([]any)
	if !ok {
		t.Fatal("response schema has no required fields")
	}
	for _, field := range required {
		if field == "message" {
			return
		}
	}
	t.Fatal("message is not required in the response schema")
}

func TestDocsAndYAML(t *testing.T) {
	handler := NewHandler(nil, nil)
	for _, tc := range []struct {
		path     string
		contains []string
	}{
		{"/api/openapi.yaml", []string{"/api/hello", "get-hello"}},
		{"/api/docs", []string{"/api/openapi"}},
	} {
		t.Run(tc.path, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, tc.path, nil))
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", response.Code)
			}
			if tc.path == "/api/docs" && !strings.HasPrefix(response.Header().Get("Content-Type"), "text/html") {
				t.Fatalf("docs content type = %q", response.Header().Get("Content-Type"))
			}
			for _, text := range tc.contains {
				if !strings.Contains(response.Body.String(), text) {
					t.Fatalf("response is missing %q", text)
				}
			}
		})
	}
}

func object(t *testing.T, value any) map[string]any {
	t.Helper()
	result, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("expected JSON object, got %#v", value)
	}
	return result
}

func TestProductionRouting(t *testing.T) {
	handler := NewHandler(fstest.MapFS{
		"index.html":  {Data: []byte("<!doctype html><title>Homelab</title>")},
		"api/missing": {Data: []byte("must not shadow API routes")},
	}, nil)
	for _, tc := range []struct {
		method, path string
		status       int
		contains     string
	}{
		{"GET", "/", 200, "<title>Homelab</title>"},
		{"GET", "/healthz", 200, "ok"},
		{"GET", "/api/hello", 200, "Hello from Homelab"},
		{"GET", "/api/openapi.json", 200, "get-hello"},
		{"GET", "/api/openapi.yaml", 200, "get-hello"},
		{"GET", "/api/docs", 200, "/api/openapi"},
		{"GET", "/api/missing", 404, ""},
		{"GET", "/api%2Fmissing", 404, ""},
		{"POST", "/api/hello", 405, ""},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(tc.method, tc.path, nil))
			if response.Code != tc.status {
				t.Fatalf("status = %d, want %d", response.Code, tc.status)
			}
			if tc.contains != "" && !strings.Contains(response.Body.String(), tc.contains) {
				t.Fatalf("body lacks %q", tc.contains)
			}
		})
	}
}

func TestAppsEndpointExists(t *testing.T) {
	response := httptest.NewRecorder()
	NewHandler(nil, nil).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/apps", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("apps without a catalog status = %d, want 503", response.Code)
	}
}

type catalogStub struct {
	snapshot  apps.Snapshot
	err       error
	refreshed bool
}

func (c *catalogStub) Snapshot(_ context.Context, refresh bool) (apps.Snapshot, error) {
	c.refreshed = refresh
	return c.snapshot, c.err
}

func TestAppsResponseAndRefreshQuery(t *testing.T) {
	catalog := &catalogStub{snapshot: apps.Snapshot{Apps: []apps.App{{
		ID: "default/calendar-http", Name: "Calendar", URL: "https://calendar.example.test", Aliases: []string{}, Group: "Apps",
	}}, UpdatedAt: time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC), Stale: true}}
	handler := NewHandler(nil, catalog)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("GET", "/api/apps?refresh=true", nil))
	if response.Code != 200 || !catalog.refreshed {
		t.Fatalf("status %d, refresh %v", response.Code, catalog.refreshed)
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body) != 3 || body["stale"] != true || body["updatedAt"] != "2026-10-08T12:00:00Z" {
		t.Fatalf("body = %#v", body)
	}
	list, ok := body["apps"].([]any)
	if !ok || len(list) != 1 {
		t.Fatalf("apps = %#v", body["apps"])
	}
	app := object(t, list[0])
	if app["url"] != "https://calendar.example.test" || len(app) != 7 {
		t.Fatalf("app = %#v", app)
	}
	aliases, ok := app["aliases"].([]any)
	if !ok || len(aliases) != 0 {
		t.Fatalf("aliases = %#v", app["aliases"])
	}
	invalid := httptest.NewRecorder()
	handler.ServeHTTP(invalid, httptest.NewRequest("GET", "/api/apps?refresh=garbage", nil))
	if invalid.Code != 422 {
		t.Fatalf("invalid refresh status %d", invalid.Code)
	}
}

func TestAppsFailureIsSanitizedAndHealthIndependent(t *testing.T) {
	handler := NewHandler(nil, &catalogStub{err: errors.New("token=secret upstream detail")})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("GET", "/api/apps", nil))
	if response.Code != 503 || strings.Contains(response.Body.String(), "secret") || !strings.Contains(response.Body.String(), "temporarily unavailable") {
		t.Fatalf("failure response = %d %s", response.Code, response.Body.String())
	}
	health := httptest.NewRecorder()
	handler.ServeHTTP(health, httptest.NewRequest("GET", "/healthz", nil))
	if health.Code != 200 || health.Body.String() != "ok\n" {
		t.Fatalf("health = %d %s", health.Code, health.Body.String())
	}
}

func TestAppsOpenAPI(t *testing.T) {
	response := httptest.NewRecorder()
	NewHandler(nil, nil).ServeHTTP(response, httptest.NewRequest("GET", "/api/openapi.json", nil))
	var spec map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &spec); err != nil {
		t.Fatal(err)
	}
	operation := object(t, object(t, object(t, spec["paths"])["/api/apps"])["get"])
	if operation["operationId"] != "list-apps" {
		t.Fatalf("operation = %#v", operation)
	}
	parameters, ok := operation["parameters"].([]any)
	if !ok || len(parameters) != 1 {
		t.Fatalf("parameters = %#v", operation["parameters"])
	}
	refresh := object(t, parameters[0])
	if refresh["name"] != "refresh" || refresh["in"] != "query" || object(t, refresh["schema"])["type"] != "boolean" {
		t.Fatalf("refresh = %#v", refresh)
	}
	responses := object(t, operation["responses"])
	if responses["200"] == nil || responses["503"] == nil {
		t.Fatalf("responses = %#v", responses)
	}
	schemas := object(t, object(t, spec["components"])["schemas"])
	snapshot := object(t, schemas["Snapshot"])
	if typ := object(t, object(t, snapshot["properties"])["apps"])["type"]; typ != "array" {
		t.Fatalf("apps schema type = %v, want non-nullable array", typ)
	}
	app := object(t, schemas["App"])
	if typ := object(t, object(t, app["properties"])["aliases"])["type"]; typ != "array" {
		t.Fatalf("aliases schema type = %v, want non-nullable array", typ)
	}
}
