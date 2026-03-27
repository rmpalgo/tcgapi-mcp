# tcgapi-mcp

`tcgapi-mcp` is a Go MCP server for the Open TCG API documented in [`docs/PLANNING.md`](docs/PLANNING.md) and the checked-in OpenAPI contract at [`openapi/tcgtracking-openapi.yaml`](openapi/tcgtracking-openapi.yaml).

This repository currently provides the first working MCP server slice:

- a thin `main.go` composition entrypoint
- explicit constructor-based dependency injection
- stable internal domain models
- an upstream API wrapper with cache-aware HTTP helpers
- category resolution and normalization seams
- stdio MCP runtime built on the official Go SDK
- 5 MCP tools, 2 concrete resources, 4 resource templates, and 3 prompts

## Build

```bash
make build
```

## Test

```bash
make test
```

## Validate

```bash
make openapi-validate
make generate-check
```

## Run

```bash
make run
```

## Release

```bash
make release-snapshot
```

The current server runs over stdio and exposes the TCG API through MCP tools, resources, and prompts. Logging remains on stderr so stdout stays reserved for MCP traffic.

OpenAPI generation is deterministic from the checked-in snapshot at [`openapi/tcgtracking-openapi.yaml`](openapi/tcgtracking-openapi.yaml). Use `make generate` to refresh generated bindings and `make generate-check` to verify they are up to date.

Set `TCG_CACHE_DIR` to persist cache snapshots across restarts and warm 7-day TTL endpoints on startup.

Make targets default to `GOCACHE=/tmp/tcgapi-mcp-go-cache` for sandbox-friendly local builds. Override with `GO_CACHE_DIR=/your/path make ...` if needed.

The repository also includes baseline release scaffolding: [`.goreleaser.yaml`](.goreleaser.yaml), GitHub release and CodeQL workflows, and contributor/security docs. `make release-snapshot` requires `goreleaser` and a real git checkout.
