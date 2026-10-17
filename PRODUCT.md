# Product

## Register

product

## Users

Kubernetes users who want to find and open their routed applications from a
desktop browser or phone.

## Product Purpose

Make applications easy to find without remembering hostnames or maintaining a
second inventory. Discover links and presentation metadata from HTTPRoutes.
Success means recognizing an app, finding it by name or hostname, and opening
it quickly, including when a refresh fails.

## Brand Personality

Spare, familiar, and practical. App identity and clear links carry the directory.

## Anti-references

Avoid decorative dashboard metrics, guessed app categories, hard-coded app
inventories, and health indicators inferred from route discovery.

## Design Principles

- Treat Kubernetes route metadata as the source of truth.
- Keep grouped app cards compact, searchable, and easy to recognize.
- Preserve hostname aliases and use initials when an icon cannot load.
- Retain usable links and identify stale discovery after a failed refresh.
- Keep loading, empty inventory, no matches, and request failures distinct.

## Accessibility & Inclusion

Support labeled search, keyboard-reachable links, visible focus, light and dark
themes, reduced motion, and phone layouts. Keep full app names and hostnames
available when visible card text is truncated.

## Agent Access

Tools read the route-derived directory through the normal catalog cache. Results preserve aliases, literal groups, update time, and stale status. Discovery does not imply destination health; tools never modify routes or probe destinations.
