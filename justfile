api:
    go run ./cmd/homelab-launcher

frontend:
    pnpm --dir frontend dev

build:
    pnpm --dir frontend install --frozen-lockfile
    pnpm --dir frontend test
    pnpm --dir frontend check
    pnpm --dir frontend build
    rm -rf internal/web/dist
    mkdir -p internal/web/dist
    touch internal/web/dist/.keep
    cp -R frontend/dist/. internal/web/dist/
    test -s internal/web/dist/index.html
    go test ./...
    go vet ./...
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o homelab-launcher ./cmd/homelab-launcher
