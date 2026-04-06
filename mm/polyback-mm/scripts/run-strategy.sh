#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
export POLYBACK_CONFIG="${POLYBACK_CONFIG:-$ROOT/configs/develop.yaml}"
exec "$ROOT/bin/strategy" "$POLYBACK_CONFIG"
