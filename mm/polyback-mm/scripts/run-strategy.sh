#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
export POLYBACK_CONFIG="${POLYBACK_CONFIG:-$ROOT/configs/develop.yaml}"
if [[ -z "${POLYBOT_HOME:-}" && -d "$ROOT/../polybot-main" ]]; then
  export POLYBOT_HOME="$(cd "$ROOT/../polybot-main" && pwd)"
fi
exec "$ROOT/bin/strategy" "$POLYBACK_CONFIG"
