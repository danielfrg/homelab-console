package httpapi

import (
	"context"
	"io/fs"
	"log"
	"net/http"
	"path"
	"reflect"
	"strings"
	"time"

	"github.com/danielfrg/homelab-console/internal/apps"
	"github.com/danielfrg/homelab-console/internal/mcp"
	"github.com/danielfrg/homelab-console/internal/web"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

type AppCatalog interface {
	Snapshot(context.Context, bool) (apps.Snapshot, error)
}

type AppsInput struct {
	Refresh bool `query:"refresh" default:"false" doc:"Bypass the discovery cache"`
}

type AppsOutput struct {
	Body apps.Snapshot
}

func NewHandler(assets fs.FS, catalog AppCatalog, console mcp.Console, mcpHosts ...string) http.Handler {
	mux := http.NewServeMux()
	mcpHandler := mcp.NewHandler(console, mcpHosts)
	config := huma.DefaultConfig("Homelab Console API", "0.2.0")
	config.OpenAPIPath = "/api/openapi"
	config.DocsPath = "/api/docs"
	config.SchemasPath = ""
	// apps.App and cluster.App would otherwise share the schema name "App".
	config.Components.Schemas = huma.NewMapRegistry("#/components/schemas/", func(t reflect.Type, hint string) string {
		name := huma.DefaultSchemaNamer(t, hint)
		if pkg := path.Base(t.PkgPath()); pkg == "cluster" || pkg == "mcp" {
			return strings.ToUpper(pkg[:1]) + pkg[1:] + name
		}
		return name
	})

	// Keep response bodies free of Huma's optional $schema metadata.
	config.CreateHooks = nil
	api := humago.New(mux, config)

	huma.Register(api, huma.Operation{
		OperationID: "list-links",
		Method:      http.MethodGet,
		Path:        "/api/links",
		Summary:     "List app links discovered from HTTPRoutes",
		Errors:      []int{http.StatusServiceUnavailable},
	}, func(ctx context.Context, input *AppsInput) (*AppsOutput, error) {
		if catalog == nil {
			return nil, huma.Error503ServiceUnavailable("App discovery is temporarily unavailable")
		}
		snapshot, err := catalog.Snapshot(ctx, input.Refresh)
		if err != nil {
			return nil, huma.Error503ServiceUnavailable("App discovery is temporarily unavailable")
		}
		return &AppsOutput{Body: snapshot}, nil
	})

	registerConsole(api, console)

	probe := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok\n"))
	}
	mux.HandleFunc("GET /healthz", probe)
	mux.HandleFunc("GET /readyz", probe)
	static := web.Handler(assets)
	mux.Handle("GET /", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Reserve the decoded API namespace even when an escaped slash misses a mux pattern.
		if r.URL.Path == "/api" || strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		static.ServeHTTP(w, r)
	}))

	writes := allowedHosts(sameOriginWrites(mux), mcpHosts)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/mcp" {
			mcpHandler.ServeHTTP(w, r)
			return
		}
		start := time.Now()
		writes.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Microsecond))
	})
}
