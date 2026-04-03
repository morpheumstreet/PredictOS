#!/usr/bin/env python3
"""
Fetch Polymarket Gamma `/events` (active, open), persist event + market rules
and per-outcome prices (0–1 / 0–100) into SQLite.

Cron-friendly: optional scan_runs row, logs via shell redirection in cron/scan.sh.

External truth URLs: optional JSON config merges into events.external_truth_source_urls
(per-event manual edits in DB are kept unless the same event is listed in config).

Usage:
  python3 collect.py
  python3 collect.py --sources-config config/external_truth_sources.json
  bash cron/scan.sh

All data goes to a single DB: strat/alpha-rules/data/alpha_rules.sqlite
"""

from __future__ import annotations

import argparse
import json
import sqlite3
import sys
import time
import traceback
from datetime import datetime, timezone
from pathlib import Path
from typing import Any
from urllib.error import HTTPError, URLError
from urllib.parse import urlencode
from urllib.request import Request, urlopen

GAMMA_BASE = "https://gamma-api.polymarket.com"


def catalog_db_path() -> str:
    """Single SQLite file for this module (alongside collect.py)."""
    return str(Path(__file__).resolve().parent / "data" / "alpha_rules.sqlite")


def utc_now_iso() -> str:
    return datetime.now(timezone.utc).replace(microsecond=0).isoformat()


def http_get_json(url: str, *, timeout: float = 60) -> Any:
    last_err: Exception | None = None
    for attempt in range(5):
        try:
            req = Request(url, headers={"User-Agent": "PredictOS-alpha-rules/1.0"})
            with urlopen(req, timeout=timeout) as resp:
                body = resp.read().decode("utf-8")
            return json.loads(body)
        except HTTPError as e:
            last_err = e
            if e.code in (500, 502, 503, 504) and attempt < 4:
                time.sleep(0.5 * (2**attempt))
                continue
            raise
        except (URLError, TimeoutError, json.JSONDecodeError) as e:
            last_err = e
            if attempt < 4:
                time.sleep(0.5 * (2**attempt))
                continue
            raise
    assert last_err is not None
    raise last_err


def parse_json_list_field(raw: Any) -> list[Any]:
    if raw is None:
        return []
    if isinstance(raw, list):
        return raw
    if isinstance(raw, str):
        raw = raw.strip()
        if not raw:
            return []
        try:
            v = json.loads(raw)
            return v if isinstance(v, list) else []
        except json.JSONDecodeError:
            return []
    return []


def parse_url_list_from_db(raw: str | None) -> list[str]:
    out: list[str] = []
    for x in parse_json_list_field(raw):
        if x is None:
            continue
        s = str(x).strip()
        if s:
            out.append(s)
    return out


def dedupe_preserve_order(urls: list[str]) -> list[str]:
    seen: set[str] = set()
    result: list[str] = []
    for u in urls:
        if u in seen:
            continue
        seen.add(u)
        result.append(u)
    return result


def clamp_price_0_1(x: float) -> float:
    return max(0.0, min(1.0, x))


def init_db(conn: sqlite3.Connection) -> None:
    conn.executescript(
        """
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
        """
    )
    conn.commit()
    conn.execute("DROP VIEW IF EXISTS v_event_market_outcomes")
    conn.execute(
        """
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
        """
    )
    conn.commit()


def load_sources_config(path: str | None) -> dict[str, Any]:
    if not path:
        return {}
    p = Path(path)
    if not p.is_file():
        return {}
    try:
        with p.open(encoding="utf-8") as f:
            data = json.load(f)
        return data if isinstance(data, dict) else {}
    except (OSError, json.JSONDecodeError):
        return {}


def config_urls_for_event(cfg: dict[str, Any], eid: str, slug: str | None) -> list[str]:
    out: list[str] = []
    by_id = cfg.get("by_event_id")
    if isinstance(by_id, dict) and eid in by_id:
        out.extend(_as_url_list(by_id[eid]))
    by_slug = cfg.get("by_slug")
    if slug and isinstance(by_slug, dict) and slug in by_slug:
        out.extend(_as_url_list(by_slug[slug]))
    return out


def _as_url_list(v: Any) -> list[str]:
    if v is None:
        return []
    if isinstance(v, str):
        s = v.strip()
        return [s] if s else []
    if isinstance(v, list):
        return [str(x).strip() for x in v if x is not None and str(x).strip()]
    return []


def merge_external_truth_urls(
    conn: sqlite3.Connection,
    eid: str,
    slug: str | None,
    cfg: dict[str, Any],
) -> str | None:
    row = conn.execute(
        "SELECT external_truth_source_urls FROM events WHERE id = ?",
        (eid,),
    ).fetchone()
    existing = parse_url_list_from_db(row[0] if row else None)
    from_cfg = config_urls_for_event(cfg, eid, slug)
    merged = dedupe_preserve_order(existing + from_cfg)
    if not merged:
        return None
    return json.dumps(merged, ensure_ascii=False)


def fetch_events_page(
    *,
    limit: int,
    offset: int,
    active: bool,
    closed: bool,
) -> list[dict[str, Any]]:
    params = {
        "active": str(active).lower(),
        "closed": str(closed).lower(),
        "limit": str(limit),
        "offset": str(offset),
    }
    url = f"{GAMMA_BASE}/events?{urlencode(params)}"
    data = http_get_json(url, timeout=60)
    if not isinstance(data, list):
        return []
    return data


def tags_to_json(tags: Any) -> str | None:
    if not tags:
        return None
    out = []
    for t in tags:
        if not isinstance(t, dict):
            continue
        out.append(
            {
                "id": t.get("id"),
                "label": t.get("label"),
                "slug": t.get("slug"),
            }
        )
    return json.dumps(out, ensure_ascii=False) if out else None


def upsert_event(
    conn: sqlite3.Connection,
    event: dict[str, Any],
    fetched_at: str,
    external_truth_json: str | None,
    preserve_profit_flag: bool,
) -> None:
    eid = str(event.get("id", ""))
    if not eid:
        return

    tags_json = tags_to_json(event.get("tags"))
    slug = event.get("slug")
    if slug is not None:
        slug = str(slug)

    if preserve_profit_flag:
        conn.execute(
            """
            INSERT INTO events (
                id, slug, ticker, title, description, resolution_source,
                start_date, end_date, active, closed, volume, liquidity,
                tags_json, updated_at_api, fetched_at,
                external_truth_source_urls, has_profit_opportunity, last_scanned_at
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, COALESCE((SELECT has_profit_opportunity FROM events WHERE id = ?), 0), ?)
            ON CONFLICT(id) DO UPDATE SET
                slug = excluded.slug,
                ticker = excluded.ticker,
                title = excluded.title,
                description = excluded.description,
                resolution_source = excluded.resolution_source,
                start_date = excluded.start_date,
                end_date = excluded.end_date,
                active = excluded.active,
                closed = excluded.closed,
                volume = excluded.volume,
                liquidity = excluded.liquidity,
                tags_json = excluded.tags_json,
                updated_at_api = excluded.updated_at_api,
                fetched_at = excluded.fetched_at,
                external_truth_source_urls = excluded.external_truth_source_urls,
                has_profit_opportunity = events.has_profit_opportunity,
                last_scanned_at = excluded.last_scanned_at
            """,
            (
                eid,
                slug,
                event.get("ticker"),
                event.get("title"),
                event.get("description"),
                event.get("resolutionSource"),
                event.get("startDate"),
                event.get("endDate"),
                1 if event.get("active") else 0,
                1 if event.get("closed") else 0,
                _float_or_none(event.get("volume")),
                _float_or_none(event.get("liquidity")),
                tags_json,
                event.get("updatedAt"),
                fetched_at,
                external_truth_json,
                eid,
                fetched_at,
            ),
        )
    else:
        conn.execute(
            """
            INSERT INTO events (
                id, slug, ticker, title, description, resolution_source,
                start_date, end_date, active, closed, volume, liquidity,
                tags_json, updated_at_api, fetched_at,
                external_truth_source_urls, has_profit_opportunity, last_scanned_at
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?)
            ON CONFLICT(id) DO UPDATE SET
                slug = excluded.slug,
                ticker = excluded.ticker,
                title = excluded.title,
                description = excluded.description,
                resolution_source = excluded.resolution_source,
                start_date = excluded.start_date,
                end_date = excluded.end_date,
                active = excluded.active,
                closed = excluded.closed,
                volume = excluded.volume,
                liquidity = excluded.liquidity,
                tags_json = excluded.tags_json,
                updated_at_api = excluded.updated_at_api,
                fetched_at = excluded.fetched_at,
                external_truth_source_urls = excluded.external_truth_source_urls,
                last_scanned_at = excluded.last_scanned_at
            """,
            (
                eid,
                slug,
                event.get("ticker"),
                event.get("title"),
                event.get("description"),
                event.get("resolutionSource"),
                event.get("startDate"),
                event.get("endDate"),
                1 if event.get("active") else 0,
                1 if event.get("closed") else 0,
                _float_or_none(event.get("volume")),
                _float_or_none(event.get("liquidity")),
                tags_json,
                event.get("updatedAt"),
                fetched_at,
                external_truth_json,
                fetched_at,
            ),
        )


def recompute_event_profit_heuristic(conn: sqlite3.Connection, event_id: str) -> None:
    """Set has_profit_opportunity if any open market has an outcome price away from 0/1 (research signal only)."""
    row = conn.execute(
        """
        SELECT 1 FROM markets m
        JOIN market_outcomes o ON o.market_id = m.id
        WHERE m.event_id = ? AND m.closed = 0
          AND o.price > 0.02 AND o.price < 0.98
        LIMIT 1
        """,
        (event_id,),
    ).fetchone()
    conn.execute(
        "UPDATE events SET has_profit_opportunity = ? WHERE id = ?",
        (1 if row else 0, event_id),
    )


def store_event_bundle(
    conn: sqlite3.Connection,
    event: dict[str, Any],
    fetched_at: str,
    sources_cfg: dict[str, Any],
    preserve_profit_flag: bool,
) -> None:
    eid = str(event.get("id", ""))
    if not eid:
        return

    slug_val = event.get("slug")
    slug = str(slug_val) if slug_val is not None else None
    ext_json = merge_external_truth_urls(conn, eid, slug, sources_cfg)
    upsert_event(conn, event, fetched_at, ext_json, preserve_profit_flag)

    markets = event.get("markets") or []
    if not isinstance(markets, list):
        markets = []

    for m in markets:
        if not isinstance(m, dict):
            continue
        mid = str(m.get("id", ""))
        if not mid:
            continue

        outcomes_raw = m.get("outcomes")
        prices_raw = m.get("outcomePrices")
        outcomes_list = parse_json_list_field(outcomes_raw)
        prices_list = parse_json_list_field(prices_raw)

        conn.execute(
            """
            INSERT OR REPLACE INTO markets (
                id, event_id, question, slug, condition_id, description,
                resolution_source, active, closed, end_date,
                outcomes_json, outcome_prices_json, updated_at_api, fetched_at
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
            """,
            (
                mid,
                eid,
                m.get("question"),
                m.get("slug"),
                m.get("conditionId"),
                m.get("description"),
                m.get("resolutionSource"),
                1 if m.get("active") else 0,
                1 if m.get("closed") else 0,
                m.get("endDate"),
                outcomes_raw if isinstance(outcomes_raw, str) else json.dumps(outcomes_list, ensure_ascii=False),
                prices_raw if isinstance(prices_raw, str) else json.dumps(prices_list, ensure_ascii=False),
                m.get("updatedAt"),
                fetched_at,
            ),
        )

        conn.execute("DELETE FROM market_outcomes WHERE market_id = ?", (mid,))

        n = max(len(outcomes_list), len(prices_list))
        for i in range(n):
            label = ""
            if i < len(outcomes_list) and outcomes_list[i] is not None:
                label = str(outcomes_list[i])
            price = 0.0
            if i < len(prices_list) and prices_list[i] is not None:
                try:
                    price = float(prices_list[i])
                except (TypeError, ValueError):
                    price = 0.0
            price = clamp_price_0_1(price)
            pct = round(price * 100.0, 6)

            conn.execute(
                """
                INSERT INTO market_outcomes (
                    market_id, outcome_index, outcome_label, price, price_pct, fetched_at
                ) VALUES (?, ?, ?, ?, ?, ?)
                """,
                (mid, i, label, price, pct, fetched_at),
            )

    if not preserve_profit_flag:
        recompute_event_profit_heuristic(conn, eid)


def _float_or_none(v: Any) -> float | None:
    if v is None:
        return None
    try:
        return float(v)
    except (TypeError, ValueError):
        return None


def begin_scan_run(conn: sqlite3.Connection) -> int:
    conn.execute(
        "INSERT INTO scan_runs (started_at, status, events_scanned) VALUES (?, 'running', NULL)",
        (utc_now_iso(),),
    )
    row = conn.execute("SELECT last_insert_rowid()").fetchone()
    return int(row[0]) if row else 0


def finish_scan_run(
    conn: sqlite3.Connection,
    run_id: int,
    *,
    ok: bool,
    events_scanned: int,
    error_message: str | None,
) -> None:
    conn.execute(
        """
        UPDATE scan_runs
        SET finished_at = ?, status = ?, events_scanned = ?, error_message = ?
        WHERE id = ?
        """,
        (
            utc_now_iso(),
            "ok" if ok else "error",
            events_scanned,
            error_message,
            run_id,
        ),
    )


def run_collect(
    db_path: str,
    *,
    page_limit: int,
    max_events: int | None,
    sleep_s: float,
    sources_config_path: str | None,
    preserve_profit_flag: bool,
    record_scan_run: bool,
) -> None:
    sources_cfg = load_sources_config(sources_config_path)
    conn = sqlite3.connect(db_path)
    run_id = 0
    total = 0
    try:
        init_db(conn)
        if record_scan_run:
            run_id = begin_scan_run(conn)
            conn.commit()

        offset = 0
        while True:
            if max_events is not None and total >= max_events:
                break
            batch = fetch_events_page(
                limit=page_limit,
                offset=offset,
                active=True,
                closed=False,
            )
            if not batch:
                break
            fetched_at = utc_now_iso()
            for ev in batch:
                if max_events is not None and total >= max_events:
                    break
                if isinstance(ev, dict):
                    store_event_bundle(conn, ev, fetched_at, sources_cfg, preserve_profit_flag)
                    total += 1
            conn.commit()
            if len(batch) < page_limit:
                break
            offset += page_limit
            if sleep_s > 0:
                time.sleep(sleep_s)

        if record_scan_run and run_id:
            finish_scan_run(conn, run_id, ok=True, events_scanned=total, error_message=None)
            conn.commit()
    except Exception:
        if record_scan_run and run_id:
            try:
                finish_scan_run(
                    conn,
                    run_id,
                    ok=False,
                    events_scanned=total,
                    error_message=traceback.format_exc()[-8000:],
                )
                conn.commit()
            except Exception:
                pass
        raise
    finally:
        conn.close()


def main() -> int:
    p = argparse.ArgumentParser(
        description="Gamma events → SQLite (rules, external truth URLs, profit flag, cron-ready).",
    )
    p.add_argument("--limit", type=int, default=100, help="Page size for /events (default: 100).")
    p.add_argument(
        "--max-events",
        type=int,
        default=None,
        help="Stop after storing this many events (for testing).",
    )
    p.add_argument(
        "--sleep",
        type=float,
        default=0.15,
        help="Seconds to sleep between pages (default: 0.15).",
    )
    p.add_argument(
        "--sources-config",
        default=None,
        help="JSON file with by_event_id / by_slug URL lists (see config/external_truth_sources.example.json).",
    )
    p.add_argument(
        "--preserve-profit-flag",
        action="store_true",
        help="Do not recompute has_profit_opportunity; keep DB values (use for manual overrides).",
    )
    p.add_argument(
        "--no-scan-run",
        action="store_true",
        help="Do not insert a row into scan_runs (default: record each run).",
    )
    args = p.parse_args()

    db_path = catalog_db_path()
    Path(db_path).parent.mkdir(parents=True, exist_ok=True)

    print(f"Collecting into {db_path} …", file=sys.stderr)
    run_collect(
        db_path,
        page_limit=max(1, args.limit),
        max_events=args.max_events,
        sleep_s=max(0.0, args.sleep),
        sources_config_path=args.sources_config,
        preserve_profit_flag=bool(args.preserve_profit_flag),
        record_scan_run=not args.no_scan_run,
    )
    print(
        "Done. Flat: SELECT * FROM v_event_market_outcomes LIMIT 20; "
        "Runs: SELECT * FROM scan_runs ORDER BY id DESC LIMIT 5;",
        file=sys.stderr,
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
