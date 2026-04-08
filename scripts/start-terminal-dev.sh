#!/usr/bin/env bash
# Start the PredictOS terminal dev server (UI + Bun /api/* routes).
#
# This is the "backend" for local terminal use: there is no separate API server.
# Optional extras (not started here):
#   - Polyback MM sidebar: run polyback-mm (see mm/polyback-mm/README.md), default POLYBACK_BOOTSTRAP_URL=http://127.0.0.1:8080
#   - Super Intelligence / arbitrage / agents: run Polyback Intelligence (mm/polyback-mm, :8085); terminal uses INTELLIGENCE_BASE_URL
#
# Usage from repo root:
#   bash scripts/start-terminal-dev.sh
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT/terminal"

if ! command -v bun >/dev/null 2>&1; then
  echo "error: bun is required (https://bun.sh)" >&2
  exit 1
fi

bun install
exec bash scripts/dev.sh
