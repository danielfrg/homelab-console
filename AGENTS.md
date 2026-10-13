# General

The app inventory is pulled directly from Kubernetes HTTPRoutes. Homelab

Route annotations under `launcher.example.net/` provide app names,
descriptions, groups, icons and visibility. Reuse this metadata instead of
maintaining a second app inventory.

- Do not edit the README.md file directly unless the user explicitly requests it.
- Read PRODUCT.md and DESIGN.md before changing the frontend; preserve the
  existing semantic tokens and local component conventions.
- Use Impeccable for design options and follow DESIGN.md for the UI foundation
  and interaction guidance.

Local development uses `just api` and `just frontend`. Kubernetes access uses
the current kubeconfig context or `KUBECONFIG`; credentials stay on the server.
`HOMELAB_DASHBOARD_ADDR` selects the API listen address, defaulting to
`127.0.0.1:8080`. `HOMELAB_DASHBOARD_API_TARGET` overrides the frontend dev
proxy target. Production serves the embedded frontend and Huma API from one
Go executable with a dedicated read-only ServiceAccount. Preserve the existing
Forgejo image build and Argo/manual restart release flow. Develop and review
changes locally first.

## Capabilities and Constraints

- Go discovers Gateway API v1 HTTPRoutes across namespaces through client-go.
  Keep Kubernetes access read-only and out of the frontend.
- Only valid hostnames ending in `.example.net` are included. The
  dashboard's own `homelab-dashboard-http` route and routes annotated with
  `enabled=false` are excluded.
- The first eligible hostname becomes the HTTPS app link; additional internal
  hostnames remain searchable aliases. Route namespace and name form the app ID.
- Route annotations supply presentation metadata. Missing names are derived
  from route names; missing groups use `Apps`. Apps sort by name, then ID.
- Huma documents `GET /api/apps` through OpenAPI. `refresh=true` bypasses the
  30-second discovery cache. Kubernetes requests have a five-second timeout.
- Discovery retains the last successful snapshot and marks it stale after a
  refresh failure. Keep low idle memory; there are no background watches or
  polling goroutines.
- Svelte 5, TypeScript, Tailwind 4 and local shadcn-svelte components render
  the launchpad, search and Refresh control.
- Search covers app names, descriptions, groups, primary hostnames and aliases.
  Cards show recognizable names and hosts, optional descriptions and icons,
  with initials as the icon fallback. Each card opens its app in a new tab.
- Keep loading, no apps, no search results and request failure states distinct.
  Failed refreshes preserve usable links and visibly identify stale results.
- Support desktop and phone layouts, long names, labeled controls and keyboard
  navigation. Route discovery is not an app health check.
- The approved launchpad design is in
  `docs/superpowers/specs/2026-10-08-launchpad-design.md`.

## Check

```shell
go test ./...
go vet ./...
pnpm --dir frontend check
pnpm --dir frontend build
```
