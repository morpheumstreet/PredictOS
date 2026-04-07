import { Database } from "bun:sqlite";
import { existsSync, mkdirSync } from "node:fs";
import { dirname } from "node:path";
import { getLocalRunLogDbPath } from "@/server/local-run-log-db-path";

const SCHEMA_VERSION = 1;

export interface AgentRunInsert {
  id: string;
  createdAtMs: number;
  feature: string;
  success: boolean;
  httpStatus: number | null;
  errorMessage: string | null;
  model: string | null;
  processingTimeMs: number | null;
  requestSummary: string;
  responseSummary: string;
}

export interface AgentRunRow {
  id: string;
  created_at: number;
  feature: string;
  success: number;
  http_status: number | null;
  error_message: string | null;
  model: string | null;
  processing_time_ms: number | null;
  request_summary: string;
  response_summary: string;
}

function ensureSchema(db: Database): void {
  const row = db.query("PRAGMA user_version").get() as { user_version: number };
  if (row.user_version >= SCHEMA_VERSION) {
    return;
  }

  db.exec(`
    CREATE TABLE IF NOT EXISTS agent_runs (
      id TEXT PRIMARY KEY NOT NULL,
      created_at INTEGER NOT NULL,
      feature TEXT NOT NULL,
      success INTEGER NOT NULL,
      http_status INTEGER,
      error_message TEXT,
      model TEXT,
      processing_time_ms INTEGER,
      request_summary TEXT NOT NULL,
      response_summary TEXT NOT NULL
    );
    CREATE INDEX IF NOT EXISTS idx_agent_runs_created_at ON agent_runs (created_at DESC);
    CREATE INDEX IF NOT EXISTS idx_agent_runs_feature_created_at ON agent_runs (feature, created_at DESC);
  `);
  db.exec(`PRAGMA user_version = ${SCHEMA_VERSION}`);
}

export function openRunLogDbWritable(): Database {
  const path = getLocalRunLogDbPath();
  mkdirSync(dirname(path), { recursive: true });
  const db = new Database(path, { create: true });
  ensureSchema(db);
  return db;
}

export function openRunLogDbReadonly(): Database | null {
  const path = getLocalRunLogDbPath();
  if (!existsSync(path)) {
    return null;
  }
  const db = new Database(path, { readonly: true, create: false });
  return db;
}

export function insertAgentRun(row: AgentRunInsert): void {
  const db = openRunLogDbWritable();
  try {
    db.query(
      `INSERT INTO agent_runs (
        id, created_at, feature, success, http_status, error_message, model, processing_time_ms, request_summary, response_summary
      ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
    ).run(
      row.id,
      row.createdAtMs,
      row.feature,
      row.success ? 1 : 0,
      row.httpStatus,
      row.errorMessage,
      row.model,
      row.processingTimeMs,
      row.requestSummary,
      row.responseSummary
    );
  } finally {
    db.close();
  }
}

/**
 * Never throws; logs SQLite errors to stderr so API handlers stay resilient.
 */
export function tryInsertAgentRun(row: AgentRunInsert): void {
  try {
    insertAgentRun(row);
  } catch (e) {
    console.error("[local-run-log] insert failed:", e);
  }
}

export function listAgentRuns(options: {
  feature: string | null;
  limit: number;
}): AgentRunRow[] {
  const db = openRunLogDbReadonly();
  if (!db) {
    return [];
  }
  try {
    const limit = Math.min(500, Math.max(1, options.limit));
    const feature = options.feature;
    const rows = db
      .query(
        `SELECT id, created_at, feature, success, http_status, error_message, model, processing_time_ms, request_summary, response_summary
         FROM agent_runs
         WHERE (?1 IS NULL OR feature = ?1)
         ORDER BY created_at DESC
         LIMIT ?2`
      )
      .all(feature, limit) as AgentRunRow[];
    return rows;
  } finally {
    db.close();
  }
}
