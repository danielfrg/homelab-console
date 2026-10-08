package httpapi

import (
	"encoding/json"
	"mime"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestHello(t *testing.T) {
	response := httptest.NewRecorder()
	NewHandler(nil).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/hello", nil))
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
	handler := NewHandler(nil)
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
	NewHandler(nil).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/openapi.json", nil))
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
	handler := NewHandler(nil)
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
	})
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
