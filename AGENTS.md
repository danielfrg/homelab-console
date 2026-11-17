# Agent Instructions

Homelab Console is a self-hosted console for a Kubernetes homelab: Argo CD
apps, workloads, pods, nodes, logs, restart, and a launchpad of app links.
It builds one Go binary that serves a Huma HTTP API, an MCP endpoint, and an
embedded Svelte 5 frontend. It was renamed from Homelab Launcher; route
annotations under the old `homelab-launcher/` prefix are still read.

Read `PRODUCT.md` for user intent and `DESIGN.md` before changing the frontend.

The `README.md` and `justfile` own setup and usage.
They are written for humans: avoid big restructures, but update them when changes are user facing.

## Architecture

- Go for the API in /cmd and /internal.
- TypeScript and Svelte 5 for the frontend in /frontend.
- Should build into a single binary with an embedded frontend.

## Behavior Invariants

- The only Kubernetes writes are the `kubectl.kubernetes.io/restartedAt`
  pod-template annotation on Deployments and StatefulSets, and Events. Never
  read Secrets or ConfigMaps, or return env values, args, or volume sources.
- Restart only workloads owned by an Argo CD app, never `OnDelete` workloads or
  anything annotated `homelab-console/restart: "false"`. Keep the in-progress
  and 3-per-hour limits. Writes never reinterpret an app name as a workload.
- Apps are Argo CD Applications; links come from route annotations. Never
  maintain a second app inventory.
- Preserve hostname aliases, icon fallbacks, stale snapshots, and distinct loading/error/empty states.
- Tools never probe app destinations.
- Health probes report the HTTP service independently of Kubernetes availability.
- MCP tools read the same cached snapshot as the HTTP API (only logs and
  restart go live); lists are bounded and stable, and errors are sanitized.
- Keep `internal/mcp/http.go`, `paging.go`, and `annotations.go` unchanged
  unless the transport itself changes; put new MCP helpers in new files.

## Development

Run the `just` recipes from the repo root:

- `just install`: install the frontend dependencies.
- `just api`: run the Go API (serves the frontend embedded by the last `just build`).
- `just frontend`: Vite dev server with HMR (proxies the API).
- `just check`: gofmt, `go vet`, race tests, oxfmt, oxlint, frontend tests and svelte-check.
- `just build`: build the frontend, embed it, and build the single binary.
- `just run`: build, then run the binary.
- `just image-check`: build the container image and run `scripts/container-smoke.sh` against it.

## Frontend Conventions

`DESIGN.md` owns the theme, typography, layout, and interaction rules; read it before changing the frontend.

`frontend/src/design-system/` is synced from the shared design-system repo
(`pnpm sync homelab-console` there) and is never edited by hand. Change it in
design-system, sync, and check with `pnpm sync --check`.

`main.ts` imports the synced design-system CSS first, then `app.css` last (this app has no
`palette.css`), so app rules win ties with the theme. Prefer tokens and class selectors over
`!important`. Never edit `frontend/src/design-system/` (it is synced from the design-system repo
with `pnpm sync`).
