#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export POLYBACK_CONFIG="${POLYBACK_CONFIG:-$ROOT/configs/develop.yaml}"
bash -c "cd \"$ROOT\" && make build"
exec "$ROOT/bin/intelligence" "$POLYBACK_CONFIG"
