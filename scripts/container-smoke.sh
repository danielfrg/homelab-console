#!/usr/bin/env bash
set -euo pipefail
image_ref=${1:?usage: container-smoke.sh IMAGE}
container="homelab-launcher-smoke-$$"
smoke_dir=$(mktemp -d)
cleanup() {
  docker rm -f "$container" >/dev/null 2>&1 || true
  rm -rf "$smoke_dir"
}
trap cleanup EXIT
[[ $(docker image inspect "$image_ref" --format '{{.Config.User}}') == 65532:65532 ]]
cat > "$smoke_dir/config" <<'YAML'
apiVersion: v1
kind: Config
clusters:
  - name: smoke
    cluster:
      server: https://127.0.0.1:1
contexts:
  - name: smoke
    context:
      cluster: smoke
current-context: smoke
YAML
chmod 755 "$smoke_dir"
chmod 644 "$smoke_dir/config"
docker run -d --name "$container" --read-only --tmpfs /tmp \
  -p 127.0.0.1::8080 -v "$smoke_dir:/kube:ro" -e KUBECONFIG=/kube/config "$image_ref" >/dev/null
port=$(docker port "$container" 8080/tcp | head -n 1)
base="http://$port"
ready=false
for _ in {1..60}; do
  if curl --fail --silent "$base/readyz" >/dev/null; then ready=true; break; fi
  sleep .2
done
if [[ "$ready" != true ]]; then docker logs "$container"; exit 1; fi
curl --fail --silent "$base/healthz" >/dev/null
html=$(curl --fail --silent "$base/")
asset=$(printf '%s' "$html" | sed -n 's/.*src="\([^" ]*\.js\)".*/\1/p')
[[ "$asset" == /assets/*.js ]]
curl --fail --silent "$base$asset" >/dev/null
curl --fail --silent "$base/api/docs" >/dev/null
curl --fail --silent "$base/api/openapi.json" >/dev/null
go run ./testdata/compat/mcp_smoke.go "$base/mcp" unavailable
printf 'Container verified: nonroot, read-only filesystem, embedded UI, API docs, and probes.\n'
