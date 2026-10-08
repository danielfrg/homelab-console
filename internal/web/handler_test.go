package web

import (
	"mime"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestStaticFiles(t *testing.T) {
	assets := fstest.MapFS{
		"index.html":     {Data: []byte("<!doctype html><title>Homelab fixture</title>")},
		"assets/app.js":  {Data: []byte("console.log('Homelab');")},
		"assets/app.css": {Data: []byte("body { color: black; }")},
		".keep":          {Data: []byte("private placeholder")},
		"assets/.source": {Data: []byte("private source")},
	}
	for _, tc := range []struct {
		method, path      string
		status            int
		contentType, body string
	}{
		{"GET", "/", 200, "text/html", "<!doctype html><title>Homelab fixture</title>"},
		{"GET", "/assets/app.js", 200, "text/javascript", "console.log('Homelab');"},
		{"GET", "/assets/app.css", 200, "text/css", "body { color: black; }"},
		{"HEAD", "/assets/app.js", 200, "text/javascript", ""},
		{"GET", "/assets/missing.js", 404, "", ""},
		{"GET", "/missing-page", 404, "", ""},
		{"GET", "/assets/", 404, "", ""},
		{"GET", "/.keep", 404, "", ""},
		{"GET", "/assets/.source", 404, "", ""},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			response := httptest.NewRecorder()
			Handler(assets).ServeHTTP(response, httptest.NewRequest(tc.method, tc.path, nil))
			if response.Code != tc.status {
				t.Fatalf("status = %d, want %d", response.Code, tc.status)
			}
			if tc.status == http.StatusOK {
				mediaType, _, err := mime.ParseMediaType(response.Header().Get("Content-Type"))
				if err != nil || mediaType != tc.contentType {
					t.Fatalf("content type = %q, error = %v", mediaType, err)
				}
				if response.Body.String() != tc.body {
					t.Fatalf("body = %q, want %q", response.Body.String(), tc.body)
				}
			}
		})
	}
}

func TestUnbuiltFrontend(t *testing.T) {
	for _, assets := range []fstest.MapFS{nil, {".keep": {Data: nil}}} {
		response := httptest.NewRecorder()
		Handler(assets).ServeHTTP(response, httptest.NewRequest("GET", "/", nil))
		if response.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404 without an index", response.Code)
		}
	}
}
