/**
 * Resolved polyback-mm public API base URL from Go-served YAML-backed config.
 * Fetches via same-origin proxy: GET /api/polyback/config/client
 */

let cache: string | null = null;

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

export async function fetchPolybackClientConfig(): Promise<
  PolybackClientConfig & { success: boolean }
> {
  const res = await fetch("/api/polyback/config/client");
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
}

/**
 * GET relay through Bun (same origin) to a specific polyback process.
 * Uses service URLs from Go client config.
 */
export async function polybackRelayJson<T>(
  target: PolybackServiceTarget,
  path: string
): Promise<{ ok: boolean; status: number; data?: T; raw: string }> {
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
