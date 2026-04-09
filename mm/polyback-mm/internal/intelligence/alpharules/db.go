package alpharules

import (
	"context"
	"database/sql"
)

const initSQL = `
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS events (
    id TEXT PRIMARY KEY,
    slug TEXT,
    ticker TEXT,
    title TEXT,
    description TEXT,
    resolution_source TEXT,
    start_date TEXT,
    end_date TEXT,
    active INTEGER NOT NULL DEFAULT 1,
    closed INTEGER NOT NULL DEFAULT 0,
    volume REAL,
    liquidity REAL,
    tags_json TEXT,
    updated_at_api TEXT,
    fetched_at TEXT NOT NULL,
    external_truth_source_urls TEXT,
    has_profit_opportunity INTEGER NOT NULL DEFAULT 0,
    last_scanned_at TEXT
);

CREATE TABLE IF NOT EXISTS markets (
    id TEXT PRIMARY KEY,
    event_id TEXT NOT NULL,
    question TEXT,
    slug TEXT,
    condition_id TEXT,
    description TEXT,
    resolution_source TEXT,
    active INTEGER NOT NULL DEFAULT 1,
    closed INTEGER NOT NULL DEFAULT 0,
    end_date TEXT,
    outcomes_json TEXT,
    outcome_prices_json TEXT,
    updated_at_api TEXT,
    fetched_at TEXT NOT NULL,
    FOREIGN KEY (event_id) REFERENCES events(id)
);

CREATE INDEX IF NOT EXISTS idx_markets_event ON markets(event_id);

CREATE TABLE IF NOT EXISTS market_outcomes (
    market_id TEXT NOT NULL,
    outcome_index INTEGER NOT NULL,
    outcome_label TEXT,
    price REAL NOT NULL,
    price_pct REAL NOT NULL,
    fetched_at TEXT NOT NULL,
    PRIMARY KEY (market_id, outcome_index),
    FOREIGN KEY (market_id) REFERENCES markets(id)
);

CREATE INDEX IF NOT EXISTS idx_outcomes_market ON market_outcomes(market_id);

CREATE TABLE IF NOT EXISTS scan_runs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    started_at TEXT NOT NULL,
    finished_at TEXT,
    status TEXT NOT NULL,
    events_scanned INTEGER,
    error_message TEXT
);
`

const viewSQL = `
DROP VIEW IF EXISTS v_event_market_outcomes;
CREATE VIEW v_event_market_outcomes AS
SELECT
    e.id AS event_id,
    e.slug AS event_slug,
    e.title AS event_title,
    e.description AS event_rules,
    e.external_truth_source_urls,
    e.has_profit_opportunity,
    e.last_scanned_at,
    m.id AS market_id,
    m.slug AS market_slug,
    m.question AS market_question,
    m.description AS market_rules,
    o.outcome_index,
    o.outcome_label,
    o.price AS price_0_1,
    o.price_pct AS price_0_100
FROM events e
JOIN markets m ON m.event_id = e.id
JOIN market_outcomes o ON o.market_id = m.id
`

// InitDB creates alpha-rules tables and the v_event_market_outcomes view (same as collect.py).
func InitDB(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, initSQL); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, viewSQL); err != nil {
		return err
	}
	return nil
}

// BeginScanRun inserts a running scan_runs row and returns its id.
func BeginScanRun(ctx context.Context, db *sql.DB, startedAt string) (int64, error) {
	res, err := db.ExecContext(ctx,
		`INSERT INTO scan_runs (started_at, status, events_scanned) VALUES (?, 'running', NULL)`,
		startedAt,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// FinishScanRun updates scan_runs completion fields.
func FinishScanRun(ctx context.Context, db *sql.DB, id int64, finishedAt string, ok bool, eventsScanned int, errMsg *string) error {
	status := "ok"
	if !ok {
		status = "error"
	}
	_, err := db.ExecContext(ctx, `
		UPDATE scan_runs
		SET finished_at = ?, status = ?, events_scanned = ?, error_message = ?
		WHERE id = ?`,
		finishedAt, status, eventsScanned, errMsg, id,
	)
	return err
}
