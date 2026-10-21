# Homelab Launcher

A launchpad for your Kubernetes apps. Ready for self-hosting.

Apps are discovered from Gateway API HTTPRoutes; route annotations supply
presentation metadata, so there is no separate app inventory to maintain.

![Homelab Launcher](docs/screenshots/homepage.png)

## Installation

### Docker

For local use with a kubeconfig whose API endpoint is reachable from the
container:

```zsh
docker run -it \
  -p 127.0.0.1:8080:8080 \
  -v "${HOME}/.kube:/kube:ro" \
  -e KUBECONFIG=/kube/config \
  ghcr.io/danielfrg/homelab-launcher:latest
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
./homelab-launcher
```

Open [http://127.0.0.1:8080](http://127.0.0.1:8080). Kubernetes access uses
the current kubeconfig context; `KUBECONFIG` can select another configuration.

## Usage

Homelab Launcher lists Gateway API v1 HTTPRoutes across namespaces. Each route
becomes one app: its first valid concrete hostname forms the HTTPS link, and
additional hostnames are searchable aliases. Hostless routes and wildcard
hostnames are skipped because they do not identify a concrete URL. HTTPS is
the link convention; Gateway listener protocols are not inspected.

Names are derived from route names. Optional annotations customize
presentation:

```yaml
metadata:
  annotations:
    homelab-launcher/name: "File Browser"
    homelab-launcher/description: "Shared files"
    homelab-launcher/group: "Tools"
    homelab-launcher/icon: "mdi-folder"
```

Set `homelab-launcher/enabled: "false"` to hide any route, including Homelab
Launcher's own route. Unannotated routes are included, with group `Apps`.
Icons use dashboard-icons names or `mdi-` names for Material Design icons.

Search covers names, descriptions, groups, primary hostnames, and aliases.
Discovery caches results for 30 seconds and keeps the last successful snapshot
when a refresh fails; Retry bypasses the cache. Discovery does not check app
health. API documentation is at `/api/docs`.

### MCP

Agents can browse the directory through the MCP endpoint at `/mcp`
(Streamable HTTP). Tools: `search_apps` and `get_app`.

MCP only accepts requests addressed to localhost. Behind a proxy or container
network, set `HOMELAB_LAUNCHER_MCP_ALLOWED_HOSTS` to your public hostname.

### Security

There is no authentication: anyone who can reach the service or the MCP
endpoint can browse the directory, so run it on a trusted network or behind an
authenticated proxy. Kubernetes access stays read-only.

## Configuration

| Variable                             | Default                                      |
| ------------------------------------ | -------------------------------------------- |
| `HOMELAB_LAUNCHER_ADDR`              | `127.0.0.1:8080`                             |
| `HOMELAB_LAUNCHER_MCP_ALLOWED_HOSTS` | unset (MCP allows localhost only)            |
| `HOMELAB_LAUNCHER_API_TARGET`        | `http://127.0.0.1:8080` (frontend dev proxy) |
| `HOMELAB_LAUNCHER_FRONTEND_PORT`     | `5173` (frontend dev server)                 |
| `KUBECONFIG`                         | standard kubeconfig loading                  |

Liveness is `/healthz`; readiness is `/readyz`. Both check the running HTTP
service independently of Kubernetes availability, keeping stale links usable
during cluster API outages.

## Development

Requires Go 1.25, Node 26, pnpm 10, and just.

The API needs a working Kubernetes context with permission to list HTTPRoutes.
If your credentials are outside the default kubeconfig, or another project
uses the default ports, create a root `.env` file (ignored by Git) that `just`
loads for both commands:

```dotenv
KUBECONFIG=/path/to/kubeconfig
HOMELAB_LAUNCHER_ADDR=127.0.0.1:8081
HOMELAB_LAUNCHER_API_TARGET=http://127.0.0.1:8081
HOMELAB_LAUNCHER_FRONTEND_PORT=5174
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
