/**
 * Resolved polyback-mm public API base URL from Go-served YAML-backed config.
 * Fetches via same-origin proxy: GET /api/polyback/config/client
 */

let cache: string | null = null;

export type PolybackClientConfig = {
  apiBaseUrl: string;
  hftMode?: string;
  server?: Record<string, unknown>;
  modules?: Array<{ name: string; pathPrefix: string }>;
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
