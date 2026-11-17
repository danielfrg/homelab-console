# Design System

Homelab Console is a simple self-hosted application.

The interface should use a small consistent vocabulary rather than
one-off styling for each new control.

## Component Rules

The theme (Cadence), the table, the theme toggle and the status badge come from the shared
design system, copied into `frontend/src/design-system/` by `pnpm sync` in the design-system repo.
That folder is never edited by hand: change it in design-system and sync.

- Controls come from `@ui-sdk/svelte` (pinned exactly). Follow the ui-sdk skill and docs before
  using one; use `Button`, `Input`, `SegmentedControl`, `Dialog` and `AlertDialog`, not bare
  `<button>` or `<input>` elements.
- Use Svelte 5 runes, typed props, and callback props.
- `app.css` loads before the theme and starts with `@layer ui-sdk;`. It holds layout and
  console-specific pieces only: it may adjust a control's layout (width, spacing), never its look.
  Raise specificity (`.page .thing`) instead of using `!important`.
- Use tokens (`--ink`, `--muted`, `--line`, `--space-*`, `--radius-*`). `--mono` is for logs
  and identifiers. No fonts are loaded or embedded: the theme uses the system font stack.
- Style custom wrappers with a class, not `data-slot` (ui-sdk sets `data-slot` itself).
- Modals use the local `Modal` (a ui-sdk `Dialog`); destructive confirmations use `AlertDialog`.
- Keep local components for app-specific behavior, not pass-through wrappers.
- Preserve `scrollbar-gutter: stable`. Keep focus visible and pressed controls stationary.

## Appearance and Accessibility

Start in dark mode and respect explicit light or system preferences. The header ThemeToggle saves
the choice under the shared `color-scheme` localStorage key, and the `preferencesScript` Vite
plugin applies it before first paint. System mode follows the operating system. Keep controls
labeled and reduced-motion preferences respected. Phone layouts must not widen the page.

## Application Patterns

Use an 88px header and begin the content at 128px on desktop, or 110px at 640px
and below. Center the main container with a 1344px maximum width and 32px
desktop side padding.

Keep the compact wordmark, theme control, search, and grouped portrait grid.
The grid fills the available width with at least two columns, adding columns
to keep 2:3 posters at most 250px tall (about 167px wide). Cards share subtle
borders; hover changes the surface and highlights all four edges, including
the rounded outer grid boundary, without motion or elevation.

Each card is one new-tab link with an icon or initials, app name, and split
hostname. Descriptions remain searchable and available as tooltips. Preserve
full accessible names and hosts even when visible text truncates.

Use literal metadata groups. Favorites comes first only when that group exists;
the default Apps group has no visible heading. Keep Retry with error or stale
notices rather than adding a permanent Refresh action. Route discovery is not an
app health check.

## Console Views

The UI stays minimal: the MCP tools are the main interface, and Grafana owns
dashboards, so there is no overview page. The header holds the wordmark, a
segmented view switcher (Apps, Pods), and the theme control; on phones the
switcher wraps to its own row. Hash routes: `#/` (apps), `#/apps/<name>`,
`#/pods`.

- Apps: the launcher grid. One card per HTTPRoute link, grouped by its literal
  route group, plus one card per Argo CD app without a link under "Without
  links". The card opens the app page; the hostname opens the site in a new
  tab. Two small dots show the owning app's sync and health; hover for the
  words. Workload counts and deploy times stay on the app page.
- App page: "Apps / name" as one heading at the same size, sync and health
  chips, one row of facts (last deploy, last sync, links), then one section per
  workload with its pods and a Restart button. The workload a search matched
  is outlined.
- Pods: one table across namespaces with a Problems filter. Pod tables are sortable and show
  pod, status, restarts (hover for the last termination), age, and Logs.

Status uses the shared `StatusBadge` tones on the app and pod pages: success for Synced/Healthy,
warning for OutOfSync, danger for Degraded or a problem, info for progressing.
Wide tables scroll inside their bordered panel; the page never scrolls
sideways. Restart always confirms in an `AlertDialog` with an optional reason; an
unrestartable workload shows a disabled button whose title explains why. Logs
open in a wide `Modal` with Reload, Previous crash (when the pod has one), and
Close.
