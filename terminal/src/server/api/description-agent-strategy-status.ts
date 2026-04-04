import { Database } from "bun:sqlite";
import { unlink } from "fs/promises";
import { dirname, join } from "path";
import { getAlphaRulesDbPath } from "@/server/alpha-rules-db-path";

function json(data: unknown, status = 200) {
  return Response.json(data, { status });
}

type RunStateFile = {
  version?: number;
  pid?: number;
  started_at?: string;
  workers?: number;
  total_jobs?: number;
  completed_jobs?: number;
  by_template?: Record<string, { queued?: number; completed?: number }>;
};

function alphaRulesRootFromDb(dbPath: string): string {
  return join(dirname(dbPath), "..");
}

function pidIsAlive(pid: number): boolean {
  if (!Number.isFinite(pid) || pid <= 0) return false;
  try {
    process.kill(pid, 0);
    return true;
  } catch {
    return false;
  }
}

async function readRunState(
  dbPath: string
): Promise<{ path: string; staleRemoved: boolean; state: RunStateFile | null }> {
  const path = join(dirname(dbPath), "description_agent_run.json");
  const f = Bun.file(path);
  if (!(await f.exists())) {
    return { path, staleRemoved: false, state: null };
  }
  let raw: string;
  try {
    raw = await f.text();
  } catch {
    return { path, staleRemoved: false, state: null };
  }
  let parsed: RunStateFile;
  try {
    parsed = JSON.parse(raw) as RunStateFile;
  } catch {
    try {
      await Bun.write(path + ".bad", raw);
      await unlink(path);
    } catch {
      /* ignore */
    }
    return { path, staleRemoved: true, state: null };
  }
  const pid = typeof parsed.pid === "number" ? parsed.pid : NaN;
  if (!pidIsAlive(pid)) {
    try {
      await unlink(path);
    } catch {
      /* ignore */
    }
    return { path, staleRemoved: true, state: null };
  }
  return { path, staleRemoved: false, state: parsed };
}

async function readParallelWorkersConfig(dbPath: string): Promise<number | null> {
  const cfgPath = join(alphaRulesRootFromDb(dbPath), "config", "description_agent.json");
  const f = Bun.file(cfgPath);
  if (!(await f.exists())) return null;
  try {
    const raw = await f.text();
    const o = JSON.parse(raw) as { parallel_workers?: number };
    const w = o.parallel_workers;
    if (typeof w === "number" && Number.isFinite(w) && w >= 1) return Math.floor(w);
    return null;
  } catch {
    return null;
  }
}

function isoEta(completed: number, total: number, startedAt: string): string | null {
  if (total <= 0 || completed <= 0 || completed >= total) return null;
  const t0 = Date.parse(startedAt);
  if (!Number.isFinite(t0)) return null;
  const elapsed = (Date.now() - t0) / 1000;
  if (elapsed < 1) return null;
  const rate = completed / elapsed;
  if (rate <= 0) return null;
  const remaining = (total - completed) / rate;
  return new Date(Date.now() + remaining * 1000).toISOString().replace(/\.\d{3}Z$/, "Z");
}

/** GET /api/description-agent-strategy-status?id=... */
export async function GET(req: Request): Promise<Response> {
  const url = new URL(req.url);
  const id = url.searchParams.get("id")?.trim();
  if (!id) {
    return json({ success: false, error: "Missing id query parameter" }, 400);
  }

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
  try {
    const strat = db
      .query("SELECT enabled FROM description_agent_strategies WHERE id = ?")
      .get(id) as { enabled: number } | undefined;
    if (!strat) {
      return json({ success: false, error: "Strategy not found" }, 404);
    }

    const workersDefault = await readParallelWorkersConfig(dbPath);
    const run = await readRunState(dbPath);

    let totalInDb = 0;
    let failedInDb = 0;
    let lastProcessedAt: string | null = null;
    try {
      db.run(
        "CREATE INDEX IF NOT EXISTS idx_agent_results_template ON description_agent_results (template_id)"
      );
      const row = db
        .query(
          `SELECT
            COUNT(*) AS n,
            SUM(CASE WHEN error IS NOT NULL AND TRIM(error) != '' THEN 1 ELSE 0 END) AS fails,
            MAX(processed_at) AS last_at
          FROM description_agent_results
          WHERE template_id = ?`
        )
        .get(id) as { n: number | null; fails: number | null; last_at: string | null } | undefined;
      if (row) {
        totalInDb = Number(row.n ?? 0);
        failedInDb = Number(row.fails ?? 0);
        lastProcessedAt = row.last_at && String(row.last_at).trim() ? String(row.last_at) : null;
      }
    } catch {
      /* table may not exist yet */
    }

    const st = run.state;
    const tpl = st?.by_template?.[id];
    const queuedInRun = tpl?.queued ?? 0;
    const completedInRun = tpl?.completed ?? 0;
    const running =
      Boolean(st) &&
      typeof st?.total_jobs === "number" &&
      st.total_jobs > 0 &&
      (st.completed_jobs ?? 0) < st.total_jobs;

    const totalJobsGlobal = st?.total_jobs ?? 0;
    const completedGlobal = st?.completed_jobs ?? 0;
    const eta =
      running && st?.started_at
        ? isoEta(completedGlobal, totalJobsGlobal, st.started_at)
        : null;

    const workersEffective = st?.workers ?? workersDefault ?? 4;

    return json({
      success: true,
      strategyId: id,
      enabled: Boolean(strat.enabled),
      runnerParallelWorkers: workersEffective,
      runnerConfigWorkers: workersDefault,
      runStatePath: run.path,
      staleRunFileRemoved: run.staleRemoved,
      running,
      runPid: st?.pid ?? null,
      runStartedAt: st?.started_at ?? null,
      runTotalJobs: totalJobsGlobal,
      runCompletedJobs: completedGlobal,
      queuedJobsThisStrategy: queuedInRun,
      completedJobsThisStrategyInRun: completedInRun,
      estimatedFinishAt: eta,
      processedRowsInDatabase: totalInDb,
      failedRowsInDatabase: failedInDb,
      lastProcessedAt,
    });
  } finally {
    db.close();
  }
}
