package alpharules

import (
	"context"
	"database/sql"
)

// InitAgentTables creates description_agent_* tables (same as description_agent.py).
func InitAgentTables(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS description_agent_results (
    target_type TEXT NOT NULL,
    target_id TEXT NOT NULL,
    template_id TEXT NOT NULL,
    input_hash TEXT NOT NULL,
    model TEXT,
    output_text TEXT,
    result_json TEXT,
    answer TEXT,
    error TEXT,
    processed_at TEXT NOT NULL,
    PRIMARY KEY (target_type, target_id, template_id)
)`); err != nil {
		return err
	}
	if err := migrateAgentResults(ctx, db); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `
CREATE INDEX IF NOT EXISTS idx_agent_results_processed
ON description_agent_results (processed_at)`); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `
CREATE INDEX IF NOT EXISTS idx_agent_results_answer
ON description_agent_results (answer)`); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS description_agent_strategies (
    id TEXT PRIMARY KEY,
    display_name TEXT,
    targets_json TEXT NOT NULL DEFAULT '[]',
    system_prompt TEXT NOT NULL DEFAULT '',
    user_prompt_template TEXT NOT NULL DEFAULT '',
    enabled INTEGER NOT NULL DEFAULT 1,
    sort_order INTEGER NOT NULL DEFAULT 0,
    model TEXT,
    temperature REAL,
    json_response_format INTEGER,
    notes TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
)`); err != nil {
		return err
	}
	_, err := db.ExecContext(ctx, `
CREATE INDEX IF NOT EXISTS idx_strategies_enabled_sort
ON description_agent_strategies (enabled, sort_order, id)`)
	return err
}

func migrateAgentResults(ctx context.Context, db *sql.DB) error {
	rows, err := db.QueryContext(ctx, `PRAGMA table_info(description_agent_results)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	names := make(map[string]struct{})
	for rows.Next() {
		var cid int
		var name, typ string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			return err
		}
		names[name] = struct{}{}
	}
	if _, ok := names["result_json"]; !ok {
		if _, err := db.ExecContext(ctx, `ALTER TABLE description_agent_results ADD COLUMN result_json TEXT`); err != nil {
			return err
		}
	}
	if _, ok := names["answer"]; !ok {
		if _, err := db.ExecContext(ctx, `ALTER TABLE description_agent_results ADD COLUMN answer TEXT`); err != nil {
			return err
		}
	}
	return rows.Err()
}
