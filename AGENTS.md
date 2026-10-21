# Agent Instructions

Homelab Launcher is a self-hosted launchpad for Kubernetes apps.
It builds one Go binary that serves a Huma HTTP API, an MCP endpoint, and an
embedded Svelte 5 frontend, discovering apps from Gateway API HTTPRoutes.

Read `PRODUCT.md` for user intent and `DESIGN.md` before changing the frontend.

The `README.md` and `justfile` own setup and usage.
They are written for humans: avoid big restructures, but update them when changes are user facing.

## Architecture

- Go for the API in /cmd and /internal.
- TypeScript and Svelte 5 for the frontend in /frontend.
- Should build into a single binary with an embedded frontend.

## Behavior Invariants

- Kubernetes access stays read-only on the server.
- Use route annotations for metadata and literal groups; never maintain a second app inventory.
- Preserve hostname aliases, icon fallbacks, stale snapshots, and distinct loading/error/empty states.
- Discovery is not an app health check; tools never modify routes or probe destinations.
- Health probes report the HTTP service independently of Kubernetes availability.
- MCP tools read the same cached catalog as the HTTP API; lists are bounded and stable, and errors are sanitized.

## Frontend Conventions

`DESIGN.md` owns the theme, typography, layout, and interaction rules; read it before changing the frontend.

## Deployment

Daniel (danielfrg/danrodriguez) deploys this service in his homelab; the deployment source is in `~/code/danielfrg/homelab`.
