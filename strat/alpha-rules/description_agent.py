#!/usr/bin/env python3
"""
Parallel LLM agent over event/market `description` text from alpha_rules.sqlite.

Uses strategies from the SQLite table description_agent_strategies (UI / API CRUD)
when strategies_source is auto|db|both, else or fallback to JSON config templates.
Each successful run must return JSON: {"answer":"yes"|"no","supporting_description":"..."}
(normalized and stored in result_json; answer duplicated in column `answer` for queries).

Cron-friendly; logs via cron/description_agent.sh.

Requires: OPENAI_API_KEY in the environment (Chat Completions API).

Optional URL override (environment):
  OPENAI_BASE_URL — OpenAI-compatible API root (default https://api.openai.com/v1);
    requests go to {OPENAI_BASE_URL}/chat/completions.

On startup, loads PredictOS/terminal/.env if it exists (setdefault only: keys
already in the process environment are left unchanged).

Stdlib only (matches collect.py).
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import sqlite3
import sys
import time
import traceback
from concurrent.futures import ThreadPoolExecutor, as_completed
from datetime import datetime, timezone
from pathlib import Path
from dataclasses import dataclass
from typing import Any
from urllib.error import HTTPError, URLError
from urllib.request import Request, urlopen

# Reuse canonical DB path from collect
from collect import catalog_db_path

_DEFAULT_OPENAI_BASE = "https://v7.ns.mormscan.io/v1"


def openai_chat_completions_url() -> str:
    """POST URL for OpenAI-compatible Chat Completions: {OPENAI_BASE_URL}/chat/completions."""
    base = os.environ.get("OPENAI_BASE_URL", _DEFAULT_OPENAI_BASE).strip().rstrip("/")
    return f"{base}/chat/completions"


def _parse_dotenv_line(line: str) -> tuple[str, str] | None:
    s = line.strip()
    if not s or s.startswith("#"):
        return None
    if s.startswith("export "):
        s = s[7:].lstrip()
    if "=" not in s:
        return None
    key, _, val = s.partition("=")
    key = key.strip()
    if not key:
        return None
    val = val.strip()
    if len(val) >= 2 and val[0] == val[-1] and val[0] in "\"'":
        val = val[1:-1]
    return (key, val)


def load_terminal_dotenv() -> None:
    """
    Read repo `terminal/.env` and apply entries with os.environ.setdefault.
    Path: strat/alpha-rules/ -> strat/ -> repo root / terminal/.env
    """
    repo = Path(__file__).resolve().parent.parent.parent
    path = repo / "terminal" / ".env"
    if not path.is_file():
        return
    try:
        raw = path.read_text(encoding="utf-8")
    except OSError:
        return
    if raw.startswith("\ufeff"):
        raw = raw[1:]
    for line in raw.splitlines():
        parsed = _parse_dotenv_line(line)
        if not parsed:
            continue
        k, v = parsed
        os.environ.setdefault(k, v)


# Stored JSON shape (after normalization)
RESULT_SCHEMA_HINT = (
    'Reply with a single JSON object only, no markdown, no extra text. '
    'Keys: "answer" (string, exactly "yes" or "no") and '
    '"supporting_description" (string, concise evidence from the rules text).'
)


def utc_now_iso() -> str:
    return datetime.now(timezone.utc).replace(microsecond=0).isoformat()


def init_agent_tables(conn: sqlite3.Connection) -> None:
    conn.execute(
        """
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
        )
        """
    )
    migrate_agent_tables(conn)
    conn.execute(
        """
        CREATE INDEX IF NOT EXISTS idx_agent_results_processed
        ON description_agent_results (processed_at)
        """
    )
    conn.execute(
        """
        CREATE INDEX IF NOT EXISTS idx_agent_results_answer
        ON description_agent_results (answer)
        """
    )
    conn.commit()


def migrate_agent_tables(conn: sqlite3.Connection) -> None:
    """Add result_json / answer if DB was created before those columns existed."""
    rows = conn.execute("PRAGMA table_info(description_agent_results)").fetchall()
    if not rows:
        return
    names = {r[1] for r in rows}
    if "result_json" not in names:
        conn.execute(
            "ALTER TABLE description_agent_results ADD COLUMN result_json TEXT"
        )
    if "answer" not in names:
        conn.execute("ALTER TABLE description_agent_results ADD COLUMN answer TEXT")
    conn.commit()


def init_strategies_table(conn: sqlite3.Connection) -> None:
    conn.execute(
        """
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
        """
    )
    conn.execute(
        """
        CREATE INDEX IF NOT EXISTS idx_strategies_enabled_sort
        ON description_agent_strategies (enabled, sort_order, id)
        """
    )
    conn.commit()


def _validate_template_entry(t: dict[str, Any], *, ctx: str) -> None:
    if not isinstance(t, dict):
        raise ValueError(f"{ctx}: each template must be an object")
    tid = t.get("id")
    if not tid or not isinstance(tid, str):
        raise ValueError(f"{ctx}: each template needs string id")
    if not isinstance(t.get("system"), str) or not isinstance(t.get("user"), str):
        raise ValueError(f'{ctx}: template "{tid}" needs string system and user')
    tt = t.get("targets")
    if tt is not None:
        if not isinstance(tt, list) or not tt:
            raise ValueError(
                f'{ctx}: template "{tid}" targets must be a non-empty array or omitted'
            )
        for x in tt:
            if str(x).lower() not in ("events", "markets"):
                raise ValueError(
                    f'{ctx}: template "{tid}" targets must be events and/or markets'
                )


def _strategy_row_to_template(row: tuple[Any, ...]) -> dict[str, Any]:
    rid, targets_json, system_prompt, user_prompt, m, temp, jf = row
    t: dict[str, Any] = {
        "id": str(rid),
        "system": str(system_prompt),
        "user": str(user_prompt),
    }
    raw = (targets_json or "").strip()
    if raw and raw not in ("[]", "null"):
        try:
            parsed = json.loads(raw)
        except json.JSONDecodeError:
            parsed = []
        if isinstance(parsed, list) and parsed:
            clean: list[str] = []
            for x in parsed:
                s = str(x).lower()
                if s in ("events", "markets"):
                    clean.append(s)
            if clean:
                t["targets"] = clean
    if m is not None and str(m).strip():
        t["model"] = str(m).strip()
    if temp is not None:
        t["temperature"] = float(temp)
    if jf is not None:
        t["json_response_format"] = bool(jf)
    return t


def load_enabled_strategies_as_templates(conn: sqlite3.Connection) -> list[dict[str, Any]]:
    init_strategies_table(conn)
    cur = conn.execute(
        """
        SELECT id, targets_json, system_prompt, user_prompt_template,
               model, temperature, json_response_format
        FROM description_agent_strategies
        WHERE enabled = 1
        ORDER BY sort_order ASC, id ASC
        """
    )
    return [_strategy_row_to_template(r) for r in cur.fetchall()]


def resolve_strategies(conn: sqlite3.Connection, cfg: dict[str, Any]) -> list[dict[str, Any]]:
    """
    strategies_source: auto (default) — DB enabled rows if any, else file templates.
    file — JSON templates only. db — DB only. both — file order, DB overrides by id, then DB-only.
    """
    src = str(cfg.get("strategies_source", "auto")).lower()
    if src not in ("auto", "file", "db", "both"):
        raise ValueError("strategies_source must be one of: auto, file, db, both")

    file_templates: list[dict[str, Any]] = []
    raw_ft = cfg.get("templates")
    if isinstance(raw_ft, list):
        file_templates = [dict(x) for x in raw_ft if isinstance(x, dict)]

    db_templates = load_enabled_strategies_as_templates(conn)

    if src == "db":
        return db_templates
    if src == "file":
        return file_templates
    if src == "both":
        by_id: dict[str, dict[str, Any]] = {t["id"]: dict(t) for t in file_templates}
        for t in db_templates:
            by_id[t["id"]] = dict(t)
        ordered: list[dict[str, Any]] = []
        seen: set[str] = set()
        for t in file_templates:
            tid = t["id"]
            ordered.append(by_id[tid])
            seen.add(tid)
        for t in db_templates:
            if t["id"] not in seen:
                ordered.append(t)
                seen.add(t["id"])
        return ordered

    if db_templates:
        return db_templates
    return file_templates


def text_hash(s: str) -> str:
    return hashlib.sha256(s.encode("utf-8")).hexdigest()


def load_config(path: str) -> dict[str, Any]:
    with open(path, encoding="utf-8") as f:
        cfg = json.load(f)
    if not isinstance(cfg, dict):
        raise ValueError("config root must be a JSON object")
    templates = cfg.get("templates")
    if templates is None:
        templates = []
    if not isinstance(templates, list):
        raise ValueError('"templates" must be an array')
    src = str(cfg.get("strategies_source", "auto")).lower()
    if src not in ("auto", "file", "db", "both"):
        raise ValueError("strategies_source must be one of: auto, file, db, both")
    if src == "file" and not templates:
        raise ValueError('strategies_source=file requires non-empty "templates" in JSON')
    for t in templates:
        _validate_template_entry(t, ctx="config JSON")
    cfg["templates"] = templates
    return cfg


def render_template(tpl: str, ctx: dict[str, Any]) -> str:
    safe = {k: "" if v is None else str(v) for k, v in ctx.items()}
    return tpl.format(**safe)


def strip_code_fence(s: str) -> str:
    t = s.strip()
    if not t.startswith("```"):
        return t
    lines = t.split("\n")
    if lines and lines[0].strip().startswith("```"):
        lines = lines[1:]
    if lines and lines[-1].strip() == "```":
        lines = lines[:-1]
    return "\n".join(lines).strip()


def normalize_yes_no(val: Any) -> str | None:
    if isinstance(val, bool):
        return "yes" if val else "no"
    if val is None:
        return None
    if not isinstance(val, str):
        return None
    s = val.strip().lower()
    if s in ("yes", "y", "true", "1"):
        return "yes"
    if s in ("no", "n", "false", "0"):
        return "no"
    return None


def parse_yes_no_json_response(raw: str) -> tuple[dict[str, Any], str]:
    """
    Parse model output into a normalized dict and canonical JSON string for SQLite.
    Raises ValueError if invalid.
    """
    cleaned = strip_code_fence(raw)
    try:
        obj = json.loads(cleaned)
    except json.JSONDecodeError as e:
        raise ValueError(f"invalid JSON: {e}") from e
    if not isinstance(obj, dict):
        raise ValueError("JSON root must be an object")

    ans = normalize_yes_no(obj.get("answer"))
    if ans is None:
        raise ValueError('missing or invalid "answer" (need yes/no)')

    support = obj.get("supporting_description")
    if support is None:
        support = obj.get("support") or obj.get("rationale") or obj.get("description")
    if not isinstance(support, str) or not support.strip():
        raise ValueError(
            'missing or empty "supporting_description" '
            "(aliases: support, rationale, description)"
        )

    normalized = {"answer": ans, "supporting_description": support.strip()}
    canonical = json.dumps(normalized, ensure_ascii=False, separators=(",", ":"))
    return normalized, canonical


def openai_chat_completion(
    *,
    api_key: str,
    model: str,
    system: str,
    user: str,
    temperature: float,
    timeout: float,
    json_object: bool,
) -> str:
    payload: dict[str, Any] = {
        "model": model,
        "temperature": temperature,
        "messages": [
            {"role": "system", "content": system},
            {"role": "user", "content": user},
        ],
    }
    if json_object:
        payload["response_format"] = {"type": "json_object"}
    body = json.dumps(payload, ensure_ascii=False).encode("utf-8")
    req = Request(
        openai_chat_completions_url(),
        data=body,
        headers={
            "Authorization": f"Bearer {api_key}",
            "Content-Type": "application/json",
        },
        method="POST",
    )
    with urlopen(req, timeout=timeout) as resp:
        data = json.loads(resp.read().decode("utf-8"))
    choices = data.get("choices") or []
    if not choices:
        raise RuntimeError(f"OpenAI response missing choices: {str(data)[:500]!r}")
    msg = choices[0].get("message") or {}
    content = msg.get("content")
    if not isinstance(content, str):
        raise RuntimeError("OpenAI response missing message.content")
    return content.strip()


def openai_with_retries(
    *,
    api_key: str,
    model: str,
    system: str,
    user: str,
    temperature: float,
    timeout: float,
    max_retries: int,
    json_object: bool,
) -> str:
    delay = 2.0
    last_err: Exception | None = None
    for attempt in range(max_retries + 1):
        try:
            return openai_chat_completion(
                api_key=api_key,
                model=model,
                system=system,
                user=user,
                temperature=temperature,
                timeout=timeout,
                json_object=json_object,
            )
        except HTTPError as e:
            last_err = e
            code = e.code
            if code in (429, 500, 502, 503) and attempt < max_retries:
                time.sleep(delay)
                delay = min(delay * 2, 60.0)
                continue
            raise
        except (URLError, TimeoutError, OSError, json.JSONDecodeError, RuntimeError) as e:
            last_err = e
            if attempt < max_retries:
                time.sleep(delay)
                delay = min(delay * 2, 60.0)
                continue
            raise
    raise RuntimeError(str(last_err))


def fetch_event_rows(conn: sqlite3.Connection) -> list[dict[str, Any]]:
    cur = conn.execute(
        """
        SELECT id, slug, title, description, resolution_source
        FROM events
        WHERE description IS NOT NULL AND TRIM(description) != ''
        """
    )
    rows = []
    for r in cur.fetchall():
        rows.append(
            {
                "target_type": "event",
                "target_id": r[0],
                "event_id": r[0],
                "event_slug": r[1],
                "event_title": r[2],
                "description": r[3],
                "resolution_source": r[4],
            }
        )
    return rows


def fetch_market_rows(conn: sqlite3.Connection) -> list[dict[str, Any]]:
    cur = conn.execute(
        """
        SELECT m.id, m.slug, m.question, m.description, m.resolution_source,
               e.id, e.slug, e.title
        FROM markets m
        JOIN events e ON e.id = m.event_id
        WHERE m.description IS NOT NULL AND TRIM(m.description) != ''
        """
    )
    rows = []
    for r in cur.fetchall():
        rows.append(
            {
                "target_type": "market",
                "target_id": r[0],
                "market_id": r[0],
                "market_slug": r[1],
                "market_question": r[2],
                "description": r[3],
                "resolution_source": r[4],
                "event_id": r[5],
                "event_slug": r[6],
                "event_title": r[7],
            }
        )
    return rows


def existing_hash(
    conn: sqlite3.Connection,
    target_type: str,
    target_id: str,
    template_id: str,
) -> str | None:
    row = conn.execute(
        """
        SELECT input_hash FROM description_agent_results
        WHERE target_type = ? AND target_id = ? AND template_id = ?
        """,
        (target_type, target_id, template_id),
    ).fetchone()
    return row[0] if row else None


def upsert_result(
    conn: sqlite3.Connection,
    *,
    target_type: str,
    target_id: str,
    template_id: str,
    input_hash: str,
    model: str,
    result_json: str | None,
    answer: str | None,
    output_text: str | None,
    error: str | None,
    processed_at: str,
) -> None:
    conn.execute(
        """
        INSERT INTO description_agent_results (
            target_type, target_id, template_id, input_hash, model,
            result_json, answer, output_text, error, processed_at
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT(target_type, target_id, template_id) DO UPDATE SET
            input_hash = excluded.input_hash,
            model = excluded.model,
            result_json = excluded.result_json,
            answer = excluded.answer,
            output_text = excluded.output_text,
            error = excluded.error,
            processed_at = excluded.processed_at
        """,
        (
            target_type,
            target_id,
            template_id,
            input_hash,
            model,
            result_json,
            answer,
            output_text,
            error,
            processed_at,
        ),
    )


@dataclass
class Job:
    target_type: str
    target_id: str
    template_id: str
    system: str
    user_rendered: str
    input_hash: str
    ctx: dict[str, Any]
    model_override: str | None = None
    temperature_override: float | None = None
    json_object_override: bool | None = None


def build_jobs(
    conn: sqlite3.Connection,
    cfg: dict[str, Any],
    *,
    targets: set[str],
    reprocess: bool,
    max_description_chars: int | None,
    max_jobs: int | None,
) -> list[Job]:
    rows: list[dict[str, Any]] = []
    if "events" in targets:
        rows.extend(fetch_event_rows(conn))
    if "markets" in targets:
        rows.extend(fetch_market_rows(conn))

    templates = cfg["templates"]
    jobs: list[Job] = []

    for row in rows:
        desc = row.get("description") or ""
        if max_description_chars is not None and len(desc) > max_description_chars:
            desc = desc[:max_description_chars] + "\n\n[truncated]"
        row = {**row, "description": desc}
        h = text_hash(desc)
        for t in templates:
            tid = t["id"]
            tmpl_targets = t.get("targets")
            if tmpl_targets is not None:
                allowed = {str(x).lower() for x in tmpl_targets}
                if row["target_type"] == "event" and "events" not in allowed:
                    continue
                if row["target_type"] == "market" and "markets" not in allowed:
                    continue
            if not reprocess:
                prev = existing_hash(conn, row["target_type"], row["target_id"], tid)
                if prev == h:
                    continue
            ctx = dict(row)
            try:
                user_rendered = render_template(t["user"], ctx)
                system_rendered = (
                    render_template(t["system"], ctx).rstrip() + "\n\n" + RESULT_SCHEMA_HINT
                )
            except KeyError as e:
                raise ValueError(
                    f'template "{tid}" references unknown placeholder: {e}'
                ) from e
            mo = t.get("model")
            mo_s = str(mo).strip() if mo is not None and str(mo).strip() else None
            to = t.get("temperature")
            to_v = float(to) if to is not None else None
            jo = t.get("json_response_format")
            jo_v = bool(jo) if jo is not None else None
            jobs.append(
                Job(
                    target_type=row["target_type"],
                    target_id=row["target_id"],
                    template_id=tid,
                    system=system_rendered,
                    user_rendered=user_rendered,
                    input_hash=h,
                    ctx=ctx,
                    model_override=mo_s,
                    temperature_override=to_v,
                    json_object_override=jo_v,
                )
            )
            if max_jobs is not None and len(jobs) >= max_jobs:
                return jobs
    return jobs


def run_worker(
    job: Job,
    *,
    api_key: str,
    model: str,
    temperature: float,
    timeout: float,
    max_retries: int,
    json_object: bool,
) -> tuple[Job, str | None, str | None, str | None, str | None]:
    """
    Returns (job, error, result_json, answer, output_text).
    On success: result_json and answer set, output_text None.
    On JSON parse failure: error set, output_text holds raw model output (truncated).
    On HTTP/API failure: error set, output_text None.
    """
    eff_model = (job.model_override or "").strip() or model
    eff_temp = (
        job.temperature_override
        if job.temperature_override is not None
        else temperature
    )
    eff_json = (
        job.json_object_override
        if job.json_object_override is not None
        else json_object
    )
    try:
        raw = openai_with_retries(
            api_key=api_key,
            model=eff_model,
            system=job.system,
            user=job.user_rendered,
            temperature=eff_temp,
            timeout=timeout,
            max_retries=max_retries,
            json_object=eff_json,
        )
        try:
            norm, canonical = parse_yes_no_json_response(raw)
            return (job, None, canonical, norm["answer"], None)
        except ValueError as e:
            err = f"JSONValidationError: {e}"
            clip = raw if len(raw) <= 8000 else raw[:8000] + "\n…[truncated]"
            return (job, err, None, None, clip)
    except Exception as e:
        err = f"{type(e).__name__}: {e}"
        return (job, err, None, None, None)


def run_agent(
    db_path: str,
    config_path: str,
    *,
    workers: int,
    batch_size: int,
    targets: set[str],
    reprocess: bool,
    max_description_chars: int | None,
    max_jobs: int | None,
    dry_run: bool,
) -> tuple[int, int, int]:
    api_key = os.environ.get("OPENAI_API_KEY", "").strip()
    if not api_key and not dry_run:
        print("OPENAI_API_KEY is not set.", file=sys.stderr)
        return (0, 0, 1)

    cfg = load_config(config_path)
    model = str(cfg.get("model") or "gpt-4.1-mini")
    temperature = float(cfg.get("temperature", 0.2))
    timeout = float(cfg.get("request_timeout_sec", 120))
    max_retries = int(cfg.get("max_retries", 2))
    json_object = bool(cfg.get("json_response_format", True))
    workers = max(1, workers)
    batch_size = max(1, batch_size)

    conn = sqlite3.connect(db_path)
    try:
        init_agent_tables(conn)
        init_strategies_table(conn)
        resolved = resolve_strategies(conn, cfg)
        cfg = {**cfg, "templates": resolved}
        if not resolved:
            print(
                "No strategies to run: enable rows in description_agent_strategies "
                "(PredictOS Agents UI) or add templates in JSON; check strategies_source.",
                file=sys.stderr,
            )
            return (0, 0, 2)
        jobs = build_jobs(
            conn,
            cfg,
            targets=targets,
            reprocess=reprocess,
            max_description_chars=max_description_chars,
            max_jobs=max_jobs,
        )
    finally:
        conn.close()

    print(
        f"Queued {len(jobs)} jobs (default_model={model}, "
        f"strategies={len(resolved)}, workers={workers}, batch_size={batch_size}).",
        file=sys.stderr,
    )
    if dry_run:
        for j in jobs[:10]:
            print(
                f"  dry-run {j.target_type} {j.target_id} template={j.template_id}",
                file=sys.stderr,
            )
        if len(jobs) > 10:
            print(f"  … and {len(jobs) - 10} more", file=sys.stderr)
        return (len(jobs), 0, 0)

    ok = 0
    fail = 0
    now = utc_now_iso()

    for i in range(0, len(jobs), batch_size):
        chunk = jobs[i : i + batch_size]
        results: list[
            tuple[Job, str | None, str | None, str | None, str | None]
        ] = []
        with ThreadPoolExecutor(max_workers=workers) as ex:
            futs = [
                ex.submit(
                    run_worker,
                    j,
                    api_key=api_key,
                    model=model,
                    temperature=temperature,
                    timeout=timeout,
                    max_retries=max_retries,
                    json_object=json_object,
                )
                for j in chunk
            ]
            for fut in as_completed(futs):
                results.append(fut.result())

        wconn = sqlite3.connect(db_path)
        try:
            init_agent_tables(wconn)
            for job, err, result_json, answer, raw_out in results:
                eff_model = (job.model_override or "").strip() or model
                upsert_result(
                    wconn,
                    target_type=job.target_type,
                    target_id=job.target_id,
                    template_id=job.template_id,
                    input_hash=job.input_hash,
                    model=eff_model,
                    result_json=result_json,
                    answer=answer,
                    output_text=raw_out if err else None,
                    error=err,
                    processed_at=now,
                )
                if err:
                    fail += 1
                    print(
                        f"FAIL {job.target_type} {job.target_id} {job.template_id}: {err}",
                        file=sys.stderr,
                    )
                else:
                    ok += 1
            wconn.commit()
        finally:
            wconn.close()

    print(f"Done. ok={ok} fail={fail}", file=sys.stderr)
    code = 1 if fail else 0
    return (ok, fail, code)


def main() -> int:
    root = Path(__file__).resolve().parent
    default_cfg = root / "config" / "description_agent.json"

    p = argparse.ArgumentParser(
        description="Run parallel LLM passes over event/market descriptions in alpha_rules.sqlite.",
    )
    p.add_argument(
        "--config",
        default=str(default_cfg),
        help=f"JSON config with templates (default: {default_cfg})",
    )
    p.add_argument(
        "--db",
        default=None,
        help="Override SQLite path (default: same as collect.py).",
    )
    p.add_argument(
        "--workers",
        type=int,
        default=None,
        help="Parallel HTTP workers (default: from config or 4).",
    )
    p.add_argument(
        "--batch-size",
        type=int,
        default=None,
        help="Jobs per DB commit batch (default: from config or 16).",
    )
    p.add_argument(
        "--targets",
        default="events,markets",
        help="Comma list: events, markets, or both (default: events,markets).",
    )
    p.add_argument(
        "--reprocess",
        action="store_true",
        help="Ignore stored input_hash; rerun all templates.",
    )
    p.add_argument(
        "--max-description-chars",
        type=int,
        default=None,
        help="Truncate description before hashing/sending (default: no limit).",
    )
    p.add_argument(
        "--max-jobs",
        type=int,
        default=None,
        help="Stop after scheduling this many jobs (testing).",
    )
    p.add_argument(
        "--dry-run",
        action="store_true",
        help="List jobs only; no API calls or DB writes.",
    )
    args = p.parse_args()
    load_terminal_dotenv()

    raw_targets = {x.strip().lower() for x in args.targets.split(",") if x.strip()}
    allowed = {"events", "markets"}
    if not raw_targets <= allowed:
        print(f"--targets must be subset of {allowed}, got {raw_targets}", file=sys.stderr)
        return 2
    if not raw_targets:
        print("--targets is empty", file=sys.stderr)
        return 2

    cfg_path = args.config
    if not Path(cfg_path).is_file():
        print(f"Config not found: {cfg_path}", file=sys.stderr)
        return 2

    cfg = load_config(cfg_path)
    workers = args.workers if args.workers is not None else int(cfg.get("parallel_workers", 4))
    batch_size = args.batch_size if args.batch_size is not None else int(cfg.get("batch_size", 16))

    db_path = args.db or catalog_db_path()
    if not Path(db_path).is_file():
        print(f"Database not found: {db_path} (run collect.py first)", file=sys.stderr)
        return 2

    try:
        _, _, code = run_agent(
            db_path,
            cfg_path,
            workers=workers,
            batch_size=batch_size,
            targets=raw_targets,
            reprocess=args.reprocess,
            max_description_chars=args.max_description_chars,
            max_jobs=args.max_jobs,
            dry_run=args.dry_run,
        )
        return code
    except (ValueError, OSError, json.JSONDecodeError) as e:
        print(f"Error: {e}", file=sys.stderr)
        return 2
    except Exception:
        traceback.print_exc()
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
