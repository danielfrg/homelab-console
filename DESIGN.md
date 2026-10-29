# Design System

Homelab Console is a simple self-hosted application.

The interface should use a small consistent vocabulary rather than
one-off styling for each new control.

## Component Rules

Use the shared desktop workspace theme in `frontend/src/theme.css`. We keep that file identical across the other self-hosted apps, so changes there should be generic and minimal.
Application layout and interaction rules belong in `frontend/src/app.css`.

- Use Svelte 5 runes, typed props, and callback props.
- Keep desktop and phone layouts, keyboard focus, and recoverable error states.
- Load Inter and IBM Plex Mono from Google Fonts with `display=swap`. Do not
embed font binaries in the repository or production assets.
- Inter is for interface text; IBM Plex Mono is for wordmarks, identifiers, and
technical data. Use neutral surfaces, subtle one-pixel borders, rounded controls,
and no decorative shadows.
- Focus is blue in light mode and amber in dark mode.
- Preserve `scrollbar-gutter: stable`.
- Import shared controls directly from `@kenn-io/kit-ui`.
- Use semantic theme tokens and keep the library's default control sizes.
- Keep local components for app-specific behavior, not pass-through wrappers.
- Give fields one visible focus treatment and keep pressed controls stationary.

## Appearance and Accessibility

Start in dark mode and respect explicit light or system preferences. The header
ThemeToggle saves the choice under the app's own storage key. System mode follows
the operating system. Keep focus visible, controls labeled, and reduced-motion
preferences respected. Phone layouts must not widen the page.

## Application Patterns

Use an 88px header and begin the content at 128px on desktop, or 110px at 640px
and below. Center the main container with a 1344px maximum width and 32px
desktop side padding, matching Golinks and CI View.

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

The header holds the wordmark, a segmented view switcher (Links, Overview, Apps,
Pods), and the theme control; on phones the switcher wraps to its own row.
Views use hash routes: `#/`, `#/overview`, `#/apps`, `#/apps/<name>`, and
`#/pods`. Links is the unchanged launcher grid.

- Overview: node cards (CPU, memory, role, kernel), then problems, apps needing
  attention, and restarts in the last 24 hours as bordered row lists.
- Apps: one table with sync and health chips, ready workloads, last deploy, and
  primary link, filterable to apps needing attention.
- App detail: status chips and facts, then one section per workload with its
  pods table and a Restart button. The workload a search matched is outlined.
- Pods: one table across namespaces with a Problems filter.

Status uses kit-ui `Chip` tones: success for Synced/Healthy, warning for
OutOfSync, danger for Degraded or a problem, info for progressing. Wide tables
scroll inside their bordered wrapper; the page itself never scrolls sideways.
Restart always confirms in a `Modal` with an optional reason; an unrestartable
workload shows a disabled button whose title explains why. Logs open in a panel
below the tables with Reload, Previous crash (when the pod has one), and Close.
This layout was built directly and reviewed in the running app rather than from
mockups.
