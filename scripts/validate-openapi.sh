#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SPEC_PATH="${1:-$ROOT_DIR/openapi/tcgtracking-openapi.yaml}"
GO_CACHE_DIR="${GO_CACHE_DIR:-/tmp/tcgapi-mcp-go-cache}"

cd "$ROOT_DIR"

env "GOCACHE=$GO_CACHE_DIR" go run ./tools/openapi-validate "$SPEC_PATH"
