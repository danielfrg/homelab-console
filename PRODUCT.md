# Product

## Users

Daniel, running a K3s homelab with Argo CD, and the agents he talks to (OpenClaw
over Telegram). He wants to find and open apps, see whether the cluster is
healthy, read why something broke, and restart it, including when he is away
from home.

Assumes deployment on a trusted network; there is no auth.

## Product Purpose

Make the homelab easy to check on and fix. Apps come from Argo CD, links from
HTTPRoutes, and live state from Kubernetes, so there is no second inventory.

Success means: "is everything OK?" gets an accurate answer, "how's frigate?"
finds Frigate inside home-assistant with its pods and warnings, "restart
frigate" restarts exactly that workload, and links still open quickly when a
refresh fails.

## Brand Personality

Spare, familiar, and practical. App identity and clear links carry the directory.

### Tone

Do not be overly explicit about explaining every part of the UI with text. It
should be intuitive. Only add explanatory messages when it's needed, such as
failed or stale discovery and empty results.

## Anti-references

Avoid decorative dashboard metrics, guessed app categories, hard-coded app
inventories, raw YAML dumps, and health inferred from route discovery. Status
comes from Kubernetes and Argo CD, and the UI should not imply an app works
just because its pods are Ready.

## Design Principles

- Treat Kubernetes route metadata as the source of truth.
- Keep grouped app cards compact, searchable, and easy to recognize.
- Preserve hostname aliases and use initials when an icon cannot load.
- Retain usable links and identify stale discovery after a failed refresh.
- Keep loading, empty inventory, no matches, and request failures distinct.
- Keep full app names and hostnames available when visible card text is
  truncated, with labeled search, keyboard-reachable links, and visible focus.

## Agent Access

Agents use six typed MCP tools over the same cached snapshot: overview, apps,
one app, pods, logs, and restart. Answers are compact enough for a chat model.
Restart is for explicit user requests only, is limited to Argo CD app
workloads, and is recorded as a Kubernetes event. Tools never probe app
destinations.
