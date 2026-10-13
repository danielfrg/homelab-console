# Homelab Dashboard

Personal launchpad for finding and opening homelab applications.

Apps are discovered directly from Kubernetes HTTPRoutes and their metadata.

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

Builds a Go excecutable with an the embedded frontend.

```shell
just build
```

## Deployment


- Forgejo Actions builds and publishes the container image.
- Uses a dedicated read-only Kubernetes ServiceAccount
