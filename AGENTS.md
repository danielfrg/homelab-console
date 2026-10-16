# homelab-launcher

Read `PRODUCT.md` for user intent and `DESIGN.md` before changing the frontend.

## Architecture

- `cmd/homelab-launcher` owns configuration, dependency wiring, and server lifecycle.
- `internal/httpapi` serves Huma endpoints; `internal/web` embeds the frontend.
- `internal/apps` discovers HTTPRoutes and caches app snapshots. Kubernetes access stays read-only on the server.
- `frontend` uses Vite, Svelte 5, TypeScript, and Kit UI with pnpm. No SvelteKit or Tailwind.

## Conventions

Import shared controls directly from `@kenn-io/kit-ui`; local components own
app-specific behavior. Use Svelte runes, typed props, and semantic CSS tokens.
Keep `frontend/src/theme.css` identical across Golinks, CI View, and Homelab Launcher.
Inter is for interface text; IBM Plex Mono is for identity and data. Preserve
`scrollbar-gutter: stable`, dark default, and explicit light/system preferences.
Keep desktop and phone layouts, keyboard focus, and recoverable error states.

Use route annotations for metadata and literal groups; never maintain a second inventory. Preserve hostname aliases, icon fallbacks, stale snapshots, and distinct loading/error/empty states. Discovery is not an app health check.

## Documentation

README sections are overview, installation, usage, configuration, development,
and build. Use synthetic examples without private paths, hostnames, or deployment
processes. Keep docs limited to screenshots and essential user references.
Keep `PRODUCT.md` and `DESIGN.md` as concise, maintained project guides.
Do not commit generated plans, specs, agent reports, or design-tool state. The root
`LICENSE` contains the standard Apache 2.0 text only.

## Development

Use `just install`, `just api`, and `just frontend`. `just check` runs Go race
tests, vet, frontend tests, and type checks. `just build` verifies and embeds the
frontend in one executable; `just image-check` builds and smoke-tests a container.
For UI changes, verify both themes and desktop/phone widths in a browser.
