import { Database } from "bun:sqlite";
import { getAlphaRulesDbPath } from "@/server/alpha-rules-db-path";

function json(data: unknown, status = 200) {
  return Response.json(data, { status });
}

const ID_RE = /^[a-zA-Z0-9][a-zA-Z0-9._-]{0,79}$/;

function isoTimestamp(): string {
  return new Date().toISOString().replace(/\.\d{3}Z$/, "Z");
}

function ensureStrategiesTable(db: Database) {
  db.run(`
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
    )
  `);
  db.run(
    "CREATE INDEX IF NOT EXISTS idx_strategies_enabled_sort ON description_agent_strategies (enabled, sort_order, id)"
  );
}

function parseTargetsJson(s: string): ("events" | "markets")[] {
  try {
    const a = JSON.parse(s || "[]");
    if (!Array.isArray(a)) return [];
    return a.filter((x): x is "events" | "markets" => x === "events" || x === "markets");
  } catch {
    return [];
  }
}

function rowToApi(r: Record<string, unknown>) {
  return {
    id: r.id,
    display_name: r.display_name ?? null,
    targets_json: r.targets_json,
    targets: parseTargetsJson(String(r.targets_json ?? "[]")),
    system_prompt: r.system_prompt,
    user_prompt_template: r.user_prompt_template,
    enabled: Boolean(r.enabled),
    sort_order: Number(r.sort_order ?? 0),
    model: r.model == null || r.model === "" ? null : String(r.model),
    temperature: r.temperature == null ? null : Number(r.temperature),
    json_response_format:
      r.json_response_format == null ? null : Boolean(r.json_response_format),
    notes: r.notes == null || r.notes === "" ? null : String(r.notes),
    created_at: r.created_at,
    updated_at: r.updated_at,
  };
}

async function openDbRw(): Promise<{ db: Database; dbPath: string } | Response> {
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
  const db = new Database(dbPath);
  ensureStrategiesTable(db);
  return { db, dbPath };
}

/** GET /api/description-agent-strategies */
export async function GET(): Promise<Response> {
  const opened = await openDbRw();
  if (opened instanceof Response) return opened;
  const { db } = opened;
  try {
    const rows = db
      .query(`SELECT * FROM description_agent_strategies ORDER BY sort_order ASC, id ASC`)
      .all() as Record<string, unknown>[];
    return json({
      success: true,
      strategies: rows.map(rowToApi),
    });
  } finally {
    db.close();
  }
}

/** POST /api/description-agent-strategies */
export async function POST(request: Request): Promise<Response> {
  const opened = await openDbRw();
  if (opened instanceof Response) return opened;
  const { db } = opened;
  try {
    let body: Record<string, unknown>;
    try {
      body = (await request.json()) as Record<string, unknown>;
    } catch {
      return json({ success: false, error: "Invalid JSON body" }, 400);
    }
    const id = typeof body.id === "string" ? body.id.trim() : "";
    if (!ID_RE.test(id)) {
      return json(
        {
          success: false,
          error:
            "Invalid id: use 1–80 chars, start with alphanumeric; allowed ._-",
        },
        400
      );
    }
    const system = typeof body.system_prompt === "string" ? body.system_prompt : "";
    const userT = typeof body.user_prompt_template === "string" ? body.user_prompt_template : "";
    if (!system.trim() || !userT.trim()) {
      return json(
        { success: false, error: "system_prompt and user_prompt_template are required" },
        400
      );
    }
    const targets = Array.isArray(body.targets) ? body.targets : [];
    const cleanTargets = targets.filter((x) => x === "events" || x === "markets") as (
      | "events"
      | "markets"
    )[];
    const targetsJson = JSON.stringify(cleanTargets);
    const displayName =
      typeof body.display_name === "string" ? body.display_name.trim() || null : null;
    const enabled = body.enabled === false ? 0 : 1;
    const sortOrder =
      typeof body.sort_order === "number" && Number.isFinite(body.sort_order)
        ? Math.floor(body.sort_order)
        : 0;
    const model =
      typeof body.model === "string" && body.model.trim() ? body.model.trim() : null;
    const temperature =
      typeof body.temperature === "number" && Number.isFinite(body.temperature)
        ? body.temperature
        : null;
    let jf: number | null = null;
    if (body.json_response_format === true) jf = 1;
    else if (body.json_response_format === false) jf = 0;
    const notes =
      typeof body.notes === "string" && body.notes.trim() ? body.notes.trim() : null;

    const now = isoTimestamp();
    const existing = db.query("SELECT id FROM description_agent_strategies WHERE id = ?").get(id);
    if (existing) {
      return json({ success: false, error: "Strategy id already exists" }, 409);
    }

    db.query(
      `INSERT INTO description_agent_strategies (
        id, display_name, targets_json, system_prompt, user_prompt_template,
        enabled, sort_order, model, temperature, json_response_format, notes,
        created_at, updated_at
      ) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`
    ).run(
      id,
      displayName,
      targetsJson,
      system,
      userT,
      enabled,
      sortOrder,
      model,
      temperature,
      jf,
      notes,
      now,
      now
    );

    const row = db.query("SELECT * FROM description_agent_strategies WHERE id = ?").get(id) as Record<
      string,
      unknown
    >;
    return json({ success: true, strategy: rowToApi(row) }, 201);
  } finally {
    db.close();
  }
}

/** PATCH /api/description-agent-strategies?id=... */
export async function PATCH(request: Request): Promise<Response> {
  const url = new URL(request.url);
  const id = url.searchParams.get("id")?.trim() ?? "";
  if (!ID_RE.test(id)) {
    return json({ success: false, error: "Missing or invalid id query param" }, 400);
  }

  const opened = await openDbRw();
  if (opened instanceof Response) return opened;
  const { db } = opened;
  try {
    let body: Record<string, unknown>;
    try {
      body = (await request.json()) as Record<string, unknown>;
    } catch {
      return json({ success: false, error: "Invalid JSON body" }, 400);
    }

    const row = db.query("SELECT * FROM description_agent_strategies WHERE id = ?").get(id) as Record<
      string,
      unknown
    > | null;
    if (!row) {
      return json({ success: false, error: "Strategy not found" }, 404);
    }

    const displayName =
      body.display_name !== undefined
        ? typeof body.display_name === "string"
          ? body.display_name.trim() || null
          : null
        : (row.display_name as string | null);
    const system =
      body.system_prompt !== undefined
        ? String(body.system_prompt)
        : String(row.system_prompt);
    const userT =
      body.user_prompt_template !== undefined
        ? String(body.user_prompt_template)
        : String(row.user_prompt_template);
    if (!system.trim() || !userT.trim()) {
      return json(
        { success: false, error: "system_prompt and user_prompt_template must be non-empty" },
        400
      );
    }

    let targetsJson = String(row.targets_json ?? "[]");
    if (body.targets !== undefined) {
      const targets = Array.isArray(body.targets) ? body.targets : [];
      const cleanTargets = targets.filter((x) => x === "events" || x === "markets") as (
        | "events"
        | "markets"
      )[];
      targetsJson = JSON.stringify(cleanTargets);
    }

    const enabled =
      body.enabled !== undefined ? (body.enabled === false ? 0 : 1) : Number(row.enabled) ? 1 : 0;
    const sortOrder =
      typeof body.sort_order === "number" && Number.isFinite(body.sort_order)
        ? Math.floor(body.sort_order)
        : Number(row.sort_order ?? 0);

    let model: string | null =
      row.model == null || row.model === "" ? null : String(row.model);
    if (body.model !== undefined) {
      model = typeof body.model === "string" && body.model.trim() ? body.model.trim() : null;
    }

    let temperature: number | null =
      row.temperature == null ? null : Number(row.temperature);
    if (body.temperature !== undefined) {
      temperature =
        typeof body.temperature === "number" && Number.isFinite(body.temperature)
          ? body.temperature
          : null;
    }

    let jf: number | null =
      row.json_response_format == null ? null : Number(row.json_response_format) ? 1 : 0;
    if (body.json_response_format === true) jf = 1;
    else if (body.json_response_format === false) jf = 0;
    else if (body.json_response_format === null) jf = null;

    let notes: string | null =
      row.notes == null || row.notes === "" ? null : String(row.notes);
    if (body.notes !== undefined) {
      notes = typeof body.notes === "string" && body.notes.trim() ? body.notes.trim() : null;
    }

    const now = isoTimestamp();
    db.query(
      `UPDATE description_agent_strategies SET
        display_name = ?, targets_json = ?, system_prompt = ?, user_prompt_template = ?,
        enabled = ?, sort_order = ?, model = ?, temperature = ?, json_response_format = ?,
        notes = ?, updated_at = ?
      WHERE id = ?`
    ).run(
      displayName,
      targetsJson,
      system,
      userT,
      enabled,
      sortOrder,
      model,
      temperature,
      jf,
      notes,
      now,
      id
    );

    const updated = db.query("SELECT * FROM description_agent_strategies WHERE id = ?").get(id) as Record<
      string,
      unknown
    >;
    return json({ success: true, strategy: rowToApi(updated) });
  } finally {
    db.close();
  }
}

/** DELETE /api/description-agent-strategies?id=... */
export async function DELETE(request: Request): Promise<Response> {
  const url = new URL(request.url);
  const id = url.searchParams.get("id")?.trim() ?? "";
  if (!ID_RE.test(id)) {
    return json({ success: false, error: "Missing or invalid id query param" }, 400);
  }

  const opened = await openDbRw();
  if (opened instanceof Response) return opened;
  const { db } = opened;
  try {
    const q = db.query("DELETE FROM description_agent_strategies WHERE id = ?");
    const info = q.run(id);
    if (info.changes === 0) {
      return json({ success: false, error: "Strategy not found" }, 404);
    }
    return json({ success: true, deleted: id });
  } finally {
    db.close();
  }
}
