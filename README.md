# Homelab Console

A small console for a Kubernetes homelab. Ready for self-hosting.

It shows Argo CD apps with their workloads, pods, and nodes, reads recent
logs, and can restart a workload, for people in the web UI and for agents
through MCP. It also keeps the launchpad: app links are discovered from Gateway
API HTTPRoutes, and route annotations supply presentation metadata, so there is
no separate app inventory to maintain.

![Homelab Console](docs/screenshots/homepage.png)

## Installation

### Docker

For local use with a kubeconfig whose API endpoint is reachable from the
container:

```zsh
docker run -it \
  -p 127.0.0.1:8080:8080 \
  -v "${HOME}/.kube:/kube:ro" \
  -e KUBECONFIG=/kube/config \
  ghcr.io/danielfrg/homelab-console:latest
```

- Open [http://localhost:8080](http://localhost:8080)
- The container runs as UID/GID 65532; mounted credentials must be readable by
  that user
- Kubeconfigs that reference external certificate files need those files
  mounted too; exec credential plugins require a custom image containing the
  plugin

In Kubernetes, use a dedicated ServiceAccount with a ClusterRole granting only
`list` on `httproutes` in `gateway.networking.k8s.io`, bound through a
ClusterRoleBinding. Mount its token for in-cluster discovery. The container
listens on port 8080.

Published images support Linux AMD64 and ARM64. `latest` tracks `main`.

### Standalone

Build a single executable with `just build` (see [Development](#development)),
then run:

```zsh
./homelab-console
```

Open [http://127.0.0.1:8080](http://127.0.0.1:8080). Kubernetes access uses
the current kubeconfig context; `KUBECONFIG` can select another configuration.

## Usage

Homelab Console lists Gateway API v1 HTTPRoutes across namespaces. Each route
becomes one app: its first valid concrete hostname forms the HTTPS link, and
additional hostnames are searchable aliases. Hostless routes and wildcard
hostnames are skipped because they do not identify a concrete URL. HTTPS is
the link convention; Gateway listener protocols are not inspected.

Names are derived from route names. Optional annotations customize
presentation:

```yaml
metadata:
  annotations:
    homelab-console/name: "File Browser"
    homelab-console/description: "Shared files"
    homelab-console/group: "Tools"
    homelab-console/icon: "mdi-folder"
```

Set `homelab-console/enabled: "false"` to hide any route, including Homelab
Console's own route. Unannotated routes are included, with group `Apps`. The
original `homelab-launcher/` prefix still works; `homelab-console/` wins when a
route has both.
Icons use dashboard-icons names or `mdi-` names for Material Design icons.

Search covers names, descriptions, groups, primary hostnames, and aliases.
Discovery caches results for 30 seconds and keeps the last successful snapshot
when a refresh fails; Retry bypasses the cache. Discovery does not check app
health. API documentation is at `/api/docs`.

### Cluster view

Apps are Argo CD Applications. Each app lists the Deployments and StatefulSets
it manages, their pods, its links, and recent warning events. Problems flag
crashing, pending, or not-ready pods, recent OOM kills and restarts, workloads
short of ready replicas, and nodes that are not Ready or under pressure.
Kubernetes status can look healthy while an app is broken, so logs are one
click (or one tool call) away.

### Restart

Restart does what `kubectl rollout restart` does. Only Deployments and
`RollingUpdate` StatefulSets that belong to an Argo CD app can be restarted.
Opt a workload or a whole app out with the annotation
`homelab-console/restart: "false"`; `OnDelete` workloads are never restarted.
A restart is refused while the previous rollout is still in progress and after
3 restarts of the same workload within an hour. Each restart is recorded as a
Kubernetes event. Set `HOMELAB_CONSOLE_RESTART_ENABLED=false` to turn restart
off.

### MCP

Agents use the MCP endpoint at `/mcp` (Streamable HTTP). Tools: `get_overview`,
`list_apps`, `get_app`, `list_pods`, `get_logs`, and `restart` (only when
restart is enabled).

MCP only accepts requests addressed to localhost. Behind a proxy or container
network, set `HOMELAB_CONSOLE_MCP_ALLOWED_HOSTS` to your public hostname.

### Security

There is no authentication: anyone who can reach the service or the MCP
endpoint can see cluster state, read pod logs, and restart workloads, so run it
on a trusted network or behind an authenticated proxy. The restart endpoint
rejects cross-site browser requests. Kubernetes writes are limited to the
restart annotation and events; never grant this service access to Secrets.

## Configuration

| Variable                             | Default                                      |
| ------------------------------------ | -------------------------------------------- |
| `HOMELAB_CONSOLE_ADDR`              | `127.0.0.1:8080`                             |
| `HOMELAB_CONSOLE_MCP_ALLOWED_HOSTS` | unset (MCP allows localhost only)            |
| `HOMELAB_CONSOLE_API_TARGET`        | `http://127.0.0.1:8080` (frontend dev proxy) |
| `HOMELAB_CONSOLE_FRONTEND_PORT`     | `5173` (frontend dev server)                 |
| `HOMELAB_CONSOLE_RESTART_ENABLED`    | `true`; only `false` turns restart off       |
| `KUBECONFIG`                         | standard kubeconfig loading                  |

Liveness is `/healthz`; readiness is `/readyz`. Both check the running HTTP
service independently of Kubernetes availability, keeping stale links usable
during cluster API outages.

## Development

Requires Go 1.25, Node 26, pnpm 10, and just.

The API needs a working Kubernetes context. With your own admin kubeconfig,
restarts from the local UI or MCP endpoint are real; set
`HOMELAB_CONSOLE_RESTART_ENABLED=false` in `.env` to browse safely.
If your credentials are outside the default kubeconfig, or another project
uses the default ports, create a root `.env` file (ignored by Git) that `just`
loads for both commands:

```dotenv
KUBECONFIG=/path/to/kubeconfig
HOMELAB_CONSOLE_ADDR=127.0.0.1:8081
HOMELAB_CONSOLE_API_TARGET=http://127.0.0.1:8081
HOMELAB_CONSOLE_FRONTEND_PORT=5174
```

Start the API and Frontend in two terminals:

```shell
# Terminal 1:
just install
just api

# Terminal 2:
just frontend
```

Open [http://127.0.0.1:5173](http://127.0.0.1:5173) (or the port from your
`.env`). Vite proxies API requests to the Go server. Run `just check` before
sending changes. See `AGENTS.md` for project structure and conventions.

## Build

```zsh
just build
```

The executable embeds the built frontend and needs no Node runtime. Build and
smoke-test a local container with:

```zsh
just image-check
```
