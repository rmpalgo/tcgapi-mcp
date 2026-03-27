#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if ! command -v goreleaser >/dev/null 2>&1; then
	printf 'goreleaser must be installed to run a snapshot release\n' >&2
	exit 1
fi

if ! git -C "$ROOT_DIR" rev-parse --show-toplevel >/dev/null 2>&1; then
	printf 'release snapshot requires a git checkout\n' >&2
	exit 1
fi

cd "$ROOT_DIR"

goreleaser release --snapshot --clean
