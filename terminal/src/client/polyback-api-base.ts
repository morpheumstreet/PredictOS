/**
 * Polyback-mm client config and relay.
 * Default: same-origin Bun proxy (/api/polyback/*).
 * Direct: set POLYBACK_BROWSER_BOOTSTRAP_URL on the terminal server and cors_allowed_origins on polyback-mm.
 */

import { isAllowedPolybackRelayPath } from "@/lib/polyback-relay-paths";

let cache: string | null = null;
let optionsCache: { browserDirectBootstrap: string | null } | null = null;

export type PolybackServiceTarget =
  | "executor"
  | "strategy"
  | "analytics"
  | "ingestor"
  | "infrastructure";

export type PolybackServiceURLs = Partial<Record<PolybackServiceTarget, string>>;

export type PolybackClientConfig = {
  apiBaseUrl: string;
  hftMode?: string;
  server?: Record<string, unknown>;
  modules?: Array<{ name: string; pathPrefix: string }>;
  serviceUrls?: PolybackServiceURLs;
};

async function getBrowserDirectBootstrap(): Promise<string | null> {
  if (optionsCache) return optionsCache.browserDirectBootstrap;
  const res = await fetch("/api/polyback/options");
  if (!res.ok) {
    return null;
  }
  const data = (await res.json()) as { browserDirectBootstrap?: string | null };
  const b =
    typeof data.browserDirectBootstrap === "string" && data.browserDirectBootstrap.trim() !== ""
      ? data.browserDirectBootstrap.trim().replace(/\/$/, "")
      : null;
  optionsCache = { browserDirectBootstrap: b };
  return b;
}

export async function fetchPolybackClientConfig(): Promise<
  PolybackClientConfig & { success: boolean }
> {
  const direct = await getBrowserDirectBootstrap();
  const url = direct
    ? `${direct}/api/v1/config/client`
    : "/api/polyback/config/client";

  const res = await fetch(url, { headers: { Accept: "application/json" } });
  const data = (await res.json()) as PolybackClientConfig & {
    success?: boolean;
    error?: string;
  };
  if (!res.ok || data.success === false || !data.apiBaseUrl) {
    throw new Error(
      typeof data.error === "string" ? data.error : "Polyback config unavailable"
    );
  }
  return data as PolybackClientConfig & { success: boolean };
}

/** Returns canonical base URL (no trailing slash), cached after first successful fetch. */
export async function getPolybackApiBaseUrl(): Promise<string> {
  if (cache) return cache;
  const cfg = await fetchPolybackClientConfig();
  cache = String(cfg.apiBaseUrl).replace(/\/$/, "");
  return cache;
}

export function clearPolybackApiBaseCache(): void {
  cache = null;
  optionsCache = null;
}

/**
 * GET relay to a polyback process. Uses Bun relay unless browser-direct mode is enabled.
 * Pass clientConfig when you already loaded it to avoid duplicate fetches (e.g. probe grid).
 */
export async function polybackRelayJson<T>(
  target: PolybackServiceTarget,
  path: string,
  opts?: { clientConfig?: PolybackClientConfig & { success: boolean } }
): Promise<{ ok: boolean; status: number; data?: T; raw: string }> {
  if (!isAllowedPolybackRelayPath(path)) {
    return {
      ok: false,
      status: 400,
      raw: JSON.stringify({ error: "path not allowed for relay" }),
    };
  }

  const direct = await getBrowserDirectBootstrap();
  if (direct) {
    const cfg = opts?.clientConfig ?? (await fetchPolybackClientConfig());
    const base = cfg.serviceUrls?.[target]?.replace(/\/$/, "");
    if (!base) {
      const raw = JSON.stringify({
        error: `No serviceUrls.${target} in client config`,
      });
      return { ok: false, status: 502, raw };
    }
    const upstream = `${base}${path}`;
    try {
      const res = await fetch(upstream, {
        headers: { Accept: "application/json" },
      });
      const raw = await res.text();
      let data: T | undefined;
      try {
        data = JSON.parse(raw) as T;
      } catch {
        /* non-JSON */
      }
      return { ok: res.ok, status: res.status, data, raw };
    } catch (e) {
      const raw = JSON.stringify({
        error: e instanceof Error ? e.message : "fetch failed",
        upstream,
      });
      return { ok: false, status: 502, raw };
    }
  }

  const q = new URLSearchParams({ target, path });
  const res = await fetch(`/api/polyback/relay?${q}`);
  const raw = await res.text();
  let data: T | undefined;
  try {
    data = JSON.parse(raw) as T;
  } catch {
    /* non-JSON */
  }
  return { ok: res.ok, status: res.status, data, raw };
}
