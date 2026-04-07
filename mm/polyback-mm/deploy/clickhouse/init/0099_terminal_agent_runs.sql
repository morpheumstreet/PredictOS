-- Terminal agent run log (exported from local SQLite via terminal/scripts/export-agent-runs-to-clickhouse.bash).
-- Source of truth: terminal/data/terminal_local.sqlite table agent_runs.
-- Append-only MergeTree for time-range analytics; deduplicate in queries if re-importing overlaps.

CREATE TABLE IF NOT EXISTS polybot.terminal_agent_runs (
  id String,
  created_at DateTime64(3),
  feature LowCardinality(String),
  success UInt8,
  http_status Nullable(UInt16),
  error_message Nullable(String),
  model Nullable(String),
  processing_time_ms Nullable(UInt32),
  request_summary String,
  response_summary String
) ENGINE = MergeTree
PARTITION BY toYYYYMM(created_at)
ORDER BY (feature, created_at, id);
