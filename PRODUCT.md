# Product

## Users

Kubernetes users who want to find and open their routed applications from a
desktop browser or phone.

Assumes deployment on a trusted network; there is no auth.

## Product Purpose

Make applications easy to find without remembering hostnames or maintaining a
second inventory. Discover links and presentation metadata from HTTPRoutes.

Success means recognizing an app, finding it by name or hostname, and opening
it quickly, including when a refresh fails.

## Brand Personality

Spare, familiar, and practical. App identity and clear links carry the directory.

### Tone

Do not be overly explicit about explaining every part of the UI with text. It
should be intuitive. Only add explanatory messages when it's needed, such as
failed or stale discovery and empty results.

## Anti-references

Avoid decorative dashboard metrics, guessed app categories, hard-coded app
inventories, and health indicators inferred from route discovery.

## Design Principles

- Treat Kubernetes route metadata as the source of truth.
- Keep grouped app cards compact, searchable, and easy to recognize.
- Preserve hostname aliases and use initials when an icon cannot load.
- Retain usable links and identify stale discovery after a failed refresh.
- Keep loading, empty inventory, no matches, and request failures distinct.
- Keep full app names and hostnames available when visible card text is
  truncated, with labeled search, keyboard-reachable links, and visible focus.

## Agent Access

Agents browse the same route-derived directory through typed MCP tools,
reading the normal catalog cache. Results preserve aliases, literal groups,
update time, and stale status. Discovery does not imply destination health;
tools never modify routes or probe destinations.
