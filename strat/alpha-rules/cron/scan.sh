#!/usr/bin/env bash
# Cron-friendly wrapper for Gamma → SQLite collection.
#
# Example crontab (run every 30 minutes UTC):
#   */30 * * * * bash /full/path/to/strat/alpha-rules/cron/scan.sh
#
# Optional: copy config/external_truth_sources.example.json to
#   config/external_truth_sources.json
# and set EXTERNAL_TRUTH_SOURCES_JSON in the environment to override path.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

DB_PATH="${POLYMARKET_SQLITE:-$ROOT/data/polymarket_events.sqlite}"
SOURCES_CFG="${EXTERNAL_TRUTH_SOURCES_JSON:-$ROOT/config/external_truth_sources.json}"
LOG_DIR="$ROOT/logs"
mkdir -p "$LOG_DIR" "$(dirname "$DB_PATH")"

TS="$(date -u +%Y%m%dT%H%M%SZ)"
LOG_FILE="$LOG_DIR/collect-${TS}.log"

EXTRA=()
if [[ -f "$SOURCES_CFG" ]]; then
  EXTRA+=(--sources-config "$SOURCES_CFG")
fi

{
  echo "=== collect start ${TS} ==="
  echo "db=${DB_PATH}"
  echo "sources_config=${SOURCES_CFG}"
  python3 "$ROOT/collect.py" --db "$DB_PATH" "${EXTRA[@]}"
  echo "=== collect end $(date -u +%Y%m%dT%H%M%SZ) ==="
} >>"$LOG_FILE" 2>&1

echo "Wrote log: $LOG_FILE"
