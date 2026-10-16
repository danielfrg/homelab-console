# Design System

## Foundation

Use the shared desktop workspace theme in `frontend/src/theme.css`. Keep that
file identical across Golinks, CI View, and Homelab Launcher. Application layout
and interaction rules belong in `frontend/src/app.css`.

Inter is for interface text; IBM Plex Mono is for wordmarks, identifiers, and
technical data. Use neutral surfaces, subtle one-pixel borders, rounded controls,
and no decorative shadows. Focus is blue in light mode and amber in dark mode.
Preserve `scrollbar-gutter: stable`.

## Component Rules

- Import shared controls directly from `@kenn-io/kit-ui`.
- Use semantic theme tokens and keep the library's default control sizes.
- Keep local components for app-specific behavior, not pass-through wrappers.
- Use Svelte 5 runes, typed props, and callback props.
- Give fields one visible focus treatment and keep pressed controls stationary.

## Appearance and Accessibility

Start in dark mode and respect explicit light or system preferences. The header
ThemeToggle saves the choice under the app's own storage key. System mode follows
the operating system. Keep focus visible, controls labeled, and reduced-motion
preferences respected. Phone layouts must not widen the page.

## Application Patterns

Keep the compact wordmark, theme control, search, and grouped portrait grid.
The grid adapts from five columns to four, three, and two. Cards share subtle
borders; hover changes the surface quietly, without motion or elevation.

Each card is one new-tab link with an icon or initials, app name, and split
hostname. Descriptions remain searchable and available as tooltips. Preserve
full accessible names and hosts even when visible text truncates.

Use literal metadata groups. Favorites comes first only when that group exists;
the default Apps group has no visible heading. Keep Retry with error or stale
notices rather than adding a permanent Refresh action. Discovery is not an app
health check.

Use `PRODUCT.md` for user intent and `README.md` for setup and commands.
