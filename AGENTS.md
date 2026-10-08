- Do not edit the README.md file directly
- Read PRODUCT.md and DESIGN.md before changing the frontend; keep the existing tokens and shared component conventions.

## Check

```shell
go test ./...
go vet ./...
pnpm --dir frontend check
pnpm --dir frontend build
```
