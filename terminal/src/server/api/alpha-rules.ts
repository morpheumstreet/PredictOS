import { Database } from "bun:sqlite";
import { getAlphaRulesDbPath } from "@/server/alpha-rules-db-path";

const TABLES = new Set([
  "events",
  "markets",
  "market_outcomes",
  "scan_runs",
  "v_event_market_outcomes",
]);

function json(data: unknown, status = 200) {
  return Response.json(data, { status });
}

function scanIntervalMinutesFromEnv(): number {
  const raw = process.env.ALPHA_RULES_SCAN_INTERVAL_MINUTES?.trim();
  const n = raw ? parseInt(raw, 10) : NaN;
  if (!Number.isFinite(n)) return 30;
  return Math.min(1440, Math.max(1, n));
}

/**
 * GET /api/alpha-rules
 * - No `table`: { success, path, exists, counts }
 * - `?table=events&limit=50&offset=0`: { success, table, rows, limit, offset, total }
 */
export async function GET(request: Request): Promise<Response> {
  const dbPath = getAlphaRulesDbPath();
  const file = Bun.file(dbPath);
  if (!(await file.exists())) {
    return json(
      {
        success: false,
        error: "alpha_rules.sqlite not found",
        path: dbPath,
        hint: "Run strat/alpha-rules/collect.py or set ALPHA_RULES_DB",
      },
      404
    );
  }

  const url = new URL(request.url);
  const table = url.searchParams.get("table")?.trim();

  const db = new Database(dbPath, { readonly: true, create: false });
  try {
    if (!table) {
      const counts: Record<string, number> = {};
      for (const name of ["events", "markets", "market_outcomes", "scan_runs"] as const) {
        const row = db.query(`SELECT COUNT(*) AS c FROM ${name}`).get() as { c: number };
        counts[name] = row.c;
      }
      const lastRun = db
        .query(
          `SELECT id, started_at, finished_at, status, events_scanned, error_message
           FROM scan_runs ORDER BY id DESC LIMIT 1`
        )
        .get();
      return json({
        success: true,
        path: dbPath,
        exists: true,
        counts,
        scanIntervalMinutes: scanIntervalMinutesFromEnv(),
        lastScanRun: lastRun ?? null,
      });
    }

    if (!TABLES.has(table)) {
      return json(
        {
          success: false,
          error: "Invalid table",
          allowed: [...TABLES],
        },
        400
      );
    }

    /** Load entire events table (up to cap) for Event Scanner UI */
    if (table === "events" && url.searchParams.get("all") === "1") {
      const MAX = 50_000;
      const total = (db.query(`SELECT COUNT(*) AS c FROM events`).get() as { c: number }).c;
      const take = Math.min(MAX, total);
      const rows = db
        .query(
          `SELECT * FROM events
           ORDER BY COALESCE(last_scanned_at, fetched_at) DESC, rowid DESC
           LIMIT ?`
        )
        .all(take);
      return json({
        success: true,
        table: "events",
        total,
        returned: rows.length,
        truncated: total > take,
        rows,
      });
    }

    const parsedLimit = parseInt(url.searchParams.get("limit") ?? "", 10);
    const limit = Number.isFinite(parsedLimit) ? Math.min(500, Math.max(1, parsedLimit)) : 50;
    const parsedOffset = parseInt(url.searchParams.get("offset") ?? "", 10);
    const offset = Number.isFinite(parsedOffset) && parsedOffset >= 0 ? parsedOffset : 0;

    if (table === "events") {
      const total = (db.query(`SELECT COUNT(*) AS c FROM events`).get() as { c: number }).c;
      const rows = db
        .query(
          `SELECT * FROM events
           ORDER BY COALESCE(last_scanned_at, fetched_at) DESC, rowid DESC
           LIMIT ? OFFSET ?`
        )
        .all(limit, offset);
      return json({
        success: true,
        table: "events",
        limit,
        offset,
        total,
        rows,
      });
    }

    const rows = db.query(`SELECT * FROM ${table} LIMIT ? OFFSET ?`).all(limit, offset);
    return json({
      success: true,
      table,
      limit,
      offset,
      rows,
    });
  } finally {
    db.close();
  }
}
