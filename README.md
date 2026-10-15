# Homelab Launcher

A launchpad for finding and opening applications discovered from Kubernetes HTTPRoutes.

Apps are discovered directly from Kubernetes HTTPRoutes and their metadata.

![Homelab Launcher screenshot](docs/screenshot.png)

## Local development

Requires Go 1.25, Node 26, pnpm 10, just, and a kubeconfig with permission to
list HTTPRoutes across namespaces:

```shell
pnpm --dir frontend install --frozen-lockfile
```

Start the API in one terminal:

```shell
just api
```

Start the UI in another:

```shell
just frontend
```

Open http://127.0.0.1:5173. The UI proxies API requests to
http://127.0.0.1:8080. Kubernetes access uses the current kubeconfig context;
`KUBECONFIG` can select a different configuration.

## Build

Builds a Go executable with the embedded frontend.

```shell
just build
```

## Discovery

The launcher lists Gateway API v1 HTTPRoutes across namespaces. Each route
becomes one app: its first valid concrete hostname forms the HTTPS link, and
additional hostnames are searchable aliases. No domain configuration, app
inventory, or network-specific defaults are needed. Hostless routes and
wildcard hostnames are skipped because they do not identify a concrete URL.
HTTPS is the link convention; Gateway listener protocols are not inspected.

Names are derived from route names. Optional annotations customize presentation:

```yaml
metadata:
  annotations:
    homelab-launcher/name: "File Browser"
    homelab-launcher/description: "Shared files"
    homelab-launcher/group: "Tools"
    homelab-launcher/icon: "mdi-folder"
```

Set `homelab-launcher/enabled: "false"` to hide any route, including the
launcher's own route. Unannotated routes are included, with group `Apps`.
Icons use dashboard-icons names or `mdi-` names for Material Design icons.

## Deployment

GitHub Actions builds public Linux AMD64 and ARM64 images at
`ghcr.io/danielfrg/homelab-launcher`. Pin a published digest in your Deployment.
The container serves the UI and API on port 8080 as a nonroot user.

Use a dedicated ServiceAccount with a ClusterRole granting only `list` on
`httproutes` in `gateway.networking.k8s.io`, bound through a ClusterRoleBinding.
The pod needs its ServiceAccount token mounted for in-cluster discovery.
Readiness and liveness probes can use `/healthz`.

`HOMELAB_LAUNCHER_ADDR` optionally changes the listen address. For local frontend
development, `HOMELAB_LAUNCHER_API_TARGET` optionally changes the API proxy target.
Neither setting configures which domains or apps are discovered.
