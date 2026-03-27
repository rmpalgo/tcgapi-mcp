#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SPEC_PATH="$ROOT_DIR/openapi/tcgtracking-openapi.yaml"
TYPES_CONFIG="$ROOT_DIR/openapi/oapi-codegen.types.yaml"
CLIENT_CONFIG="$ROOT_DIR/openapi/oapi-codegen.client.yaml"
GO_CACHE_DIR="${GO_CACHE_DIR:-/tmp/tcgapi-mcp-go-cache}"
GO=(env "GOCACHE=$GO_CACHE_DIR" go)

cd "$ROOT_DIR"

"${GO[@]}" run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen -config "$TYPES_CONFIG" "$SPEC_PATH"
"${GO[@]}" run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen -config "$CLIENT_CONFIG" "$SPEC_PATH"
