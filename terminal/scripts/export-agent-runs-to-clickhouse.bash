#!/usr/bin/env bash
# Export PredictOS terminal agent_runs (SQLite) into ClickHouse polybot.terminal_agent_runs.
#
# Prerequisites: sqlite3, clickhouse-client on PATH; CH table from
#   mm/polyback-mm/deploy/clickhouse/init/0099_terminal_agent_runs.sql
#
# Environment (all optional except as noted):
#   TERMINAL_LOCAL_DB   - SQLite path (default: <terminal>/data/terminal_local.sqlite)
#   CLICKHOUSE_HOST     - default 127.0.0.1
#   CLICKHOUSE_PORT     - native TCP port, default 9000
#   CLICKHOUSE_USER     - default default
#   CLICKHOUSE_PASSWORD - default empty (local dev)
#   CLICKHOUSE_DB       - database name, default polybot
#
# Idempotency: re-running appends duplicate ids unless you TRUNCATE the CH table first
# or query with argMax/ReplacingMergeTree; for dev, truncate before reload:
#   clickhouse-client -q "TRUNCATE TABLE polybot.terminal_agent_runs"
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TERMINAL_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

DB="${TERMINAL_LOCAL_DB:-$TERMINAL_ROOT/data/terminal_local.sqlite}"
CLICKHOUSE_HOST="${CLICKHOUSE_HOST:-127.0.0.1}"
CLICKHOUSE_PORT="${CLICKHOUSE_PORT:-9000}"
CLICKHOUSE_USER="${CLICKHOUSE_USER:-default}"
CLICKHOUSE_PASSWORD="${CLICKHOUSE_PASSWORD:-}"
CLICKHOUSE_DB="${CLICKHOUSE_DB:-polybot}"

if [[ ! -f "$DB" ]]; then
  echo "error: SQLite file not found: $DB" >&2
  echo "Set TERMINAL_LOCAL_DB or run the terminal arbitrage flow once to create the DB." >&2
  exit 1
fi

TMP_CSV="$(mktemp "${TMPDIR:-/tmp}/agent_runs_export.XXXXXX.csv")"
cleanup() { rm -f "$TMP_CSV"; }
trap cleanup EXIT

sqlite3 -csv "$DB" <<'SQL' >"$TMP_CSV"
.headers off
SELECT
  id,
  created_at,
  feature,
  success,
  ifnull(cast(http_status AS text), ''),
  ifnull(error_message, ''),
  ifnull(model, ''),
  ifnull(cast(processing_time_ms AS text), ''),
  request_summary,
  response_summary
FROM agent_runs
ORDER BY created_at;
SQL

CH_ARGS=(
  --host "$CLICKHOUSE_HOST"
  --port "$CLICKHOUSE_PORT"
  --user "$CLICKHOUSE_USER"
  --database "$CLICKHOUSE_DB"
)
if [[ -n "$CLICKHOUSE_PASSWORD" ]]; then
  CH_ARGS+=(--password "$CLICKHOUSE_PASSWORD")
fi

# Map CSV columns positionally to input(); convert created_at ms -> DateTime64(3).
clickhouse-client "${CH_ARGS[@]}" --query "$(cat <<'EOSQL'
INSERT INTO terminal_agent_runs
SELECT
  id,
  fromUnixTimestamp64Milli(toInt64(created_at_ms)),
  feature,
  toUInt8(success_i),
  if(http_status_s = '', NULL, toUInt16OrNull(http_status_s)),
  if(error_message = '', NULL, error_message),
  if(model = '', NULL, model),
  if(processing_ms_s = '', NULL, toUInt32OrNull(processing_ms_s)),
  request_summary,
  response_summary
FROM input(
  'id String, created_at_ms Int64, feature String, success_i Int64, http_status_s String, error_message String, model String, processing_ms_s String, request_summary String, response_summary String'
)
FORMAT CSV
EOSQL
)" <"$TMP_CSV"

echo "Inserted rows from $DB into ${CLICKHOUSE_DB}.terminal_agent_runs"
