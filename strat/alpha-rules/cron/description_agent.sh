#!/usr/bin/env bash
# Parallel description agent (LLM over events/markets.description).
# Runs independently of collect; safe to schedule alongside cron/scan.sh.
#
# Setup:
#   cp config/description_agent.example.json config/description_agent.json
#   OPENAI_* : put in repo terminal/.env (loaded by description_agent.py) or export for cron.
#
# Example crontab (every hour, after collect):
#   0 * * * * bash /full/path/to/strat/alpha-rules/cron/description_agent.sh

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

CFG="${DESCRIPTION_AGENT_CONFIG:-$ROOT/config/description_agent.json}"
LOG_DIR="$ROOT/logs"
mkdir -p "$LOG_DIR"

TS="$(date -u +%Y%m%dT%H%M%SZ)"
LOG_FILE="$LOG_DIR/description-agent-${TS}.log"

EXTRA=()
if [[ -f "$CFG" ]]; then
  EXTRA+=(--config "$CFG")
else
  echo "Missing config: $CFG (copy config/description_agent.example.json)" >&2
  exit 2
fi

{
  echo "=== description_agent start ${TS} ==="
  echo "db=${ROOT}/data/alpha_rules.sqlite"
  echo "config=${CFG}"
  python3 "$ROOT/description_agent.py" "${EXTRA[@]}"
  echo "=== description_agent end $(date -u +%Y%m%dT%H%M%SZ) ==="
} >>"$LOG_FILE" 2>&1

echo "Wrote log: $LOG_FILE"
