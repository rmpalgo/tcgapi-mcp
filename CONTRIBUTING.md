# Contributing

## Development Setup

- Install Go 1.25 or newer.
- Run commands from the repository root.
- Make targets default to `GOCACHE=/tmp/tcgapi-mcp-go-cache` in sandboxed environments. Override with `GO_CACHE_DIR=/your/path make ...` if you need a different cache path.

## Common Commands

```bash
make fmt
make lint
make test
make build
make openapi-validate
make generate-check
```

## OpenAPI Workflow

- The checked-in contract at `openapi/tcgtracking-openapi.yaml` is the source of truth.
- Regenerate bindings with `make generate`.
- Verify there is no generation drift with `make generate-check`.
- Keep generated code changes confined to `internal/tcgapi/generated` unless the upstream contract requires wrapper or mapping updates.

## Runtime Constraints

- Stdout is reserved for MCP traffic. Runtime logging must stay on stderr.
- The MCP layer should not import `internal/tcgapi/generated` directly.
- Prefer extending the handwritten `internal/tcgapi` wrapper and `internal/domain` models instead of leaking generated types upward.

## Pull Requests

- Keep changes focused and explain behavioral impact clearly.
- Add or update tests for behavior changes.
- Mention any environment assumptions, network requirements, or release implications in the PR description.
