package httpapi

import (
	"context"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"time"

	"homelab-dashboard/internal/web"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

type HelloBody struct {
	Message string `json:"message" doc:"Greeting from the Homelab API" example:"Hello from Homelab"`
}

type HelloOutput struct {
	Body HelloBody
}

func NewHandler(assets fs.FS) http.Handler {
	mux := http.NewServeMux()
	config := huma.DefaultConfig("Homelab API", "0.1.0")
	config.OpenAPIPath = "/api/openapi"
	config.DocsPath = "/api/docs"
	config.SchemasPath = ""

	// Keep response bodies free of Huma's optional $schema metadata.
	config.CreateHooks = nil
	api := humago.New(mux, config)

	huma.Register(api, huma.Operation{
		OperationID: "get-hello",
		Method:      http.MethodGet,
		Path:        "/api/hello",
		Summary:     "Get the Homelab greeting",
	}, func(_ context.Context, _ *struct{}) (*HelloOutput, error) {
		return &HelloOutput{Body: HelloBody{Message: "Hello from Homelab"}}, nil
	})

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok\n"))
	})
	static := web.Handler(assets)
	mux.Handle("GET /", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Reserve the decoded API namespace even when an escaped slash misses a mux pattern.
		if r.URL.Path == "/api" || strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		static.ServeHTTP(w, r)
	}))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		mux.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Microsecond))
	})
}
