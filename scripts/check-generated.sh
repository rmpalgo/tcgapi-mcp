#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GENERATED_DIR="$ROOT_DIR/internal/tcgapi/generated"
TMP_DIR="$(mktemp -d)"

cleanup() {
	rm -rf "$GENERATED_DIR"
	mkdir -p "$GENERATED_DIR"
	cp -R "$TMP_DIR/original/." "$GENERATED_DIR/"
	rm -rf "$TMP_DIR"
}

trap cleanup EXIT

if [ ! -d "$GENERATED_DIR" ]; then
	printf 'generated directory is missing: %s\n' "$GENERATED_DIR" >&2
	exit 1
fi

cp -R "$GENERATED_DIR" "$TMP_DIR/original"

"$ROOT_DIR/scripts/generate-openapi.sh"

if diff -ru "$TMP_DIR/original" "$GENERATED_DIR" >/dev/null; then
	printf 'generated code is up to date\n'
	exit 0
fi

printf 'generated code is stale; run make generate\n' >&2
diff -ru "$TMP_DIR/original" "$GENERATED_DIR" || true
exit 1
