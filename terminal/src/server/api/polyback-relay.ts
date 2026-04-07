/**
 * GET /api/polyback/relay?target=executor|strategy|...&path=/api/...
 * Forwards to the polyback service base URL from Go client config (cached).
 * Only GET; path allowlist to avoid open proxy abuse.
 */

type ServiceTarget = "executor" | "strategy" | "analytics" | "ingestor" | "infrastructure";

type CachedClientConfig = {
  at: number;
  body: Record<string, unknown>;
};

const CACHE_TTL_MS = 15_000;
let cached: CachedClientConfig | null = null;

const VALID_TARGETS: ServiceTarget[] = [
  "executor",
  "strategy",
  "analytics",
  "ingestor",
  "infrastructure",
];

const ALLOWED_PATH_PREFIXES = [
  "/api/polymarket",
  "/api/strategy",
  "/api/ingestor",
  "/api/analytics",
  "/api/infrastructure",
  "/actuator",
];

function isAllowedPath(path: string): boolean {
  if (!path.startsWith("/") || path.includes("..")) {
    return false;
  }
  return ALLOWED_PATH_PREFIXES.some((p) => path === p || path.startsWith(`${p}/`));
}

async function fetchGoClientConfig(): Promise<Record<string, unknown>> {
  if (cached && Date.now() - cached.at < CACHE_TTL_MS) {
    return cached.body;
  }
  const bootstrap = process.env.POLYBACK_BOOTSTRAP_URL?.trim() || "http://127.0.0.1:8080";
  const url = `${bootstrap.replace(/\/$/, "")}/api/v1/config/client`;
  const res = await fetch(url, { headers: { Accept: "application/json" } });
  if (!res.ok) {
    throw new Error(`client config upstream ${res.status}`);
  }
  const body = (await res.json()) as Record<string, unknown>;
  cached = { at: Date.now(), body };
  return body;
}

function baseForTarget(cfg: Record<string, unknown>, target: ServiceTarget): string | undefined {
  const urls = cfg.serviceUrls as Record<string, string> | undefined;
  if (!urls || typeof urls !== "object") {
    return undefined;
  }
  const b = urls[target];
  return typeof b === "string" && b.trim() !== "" ? b.trim().replace(/\/$/, "") : undefined;
}

export async function GET(request: Request): Promise<Response> {
  const url = new URL(request.url);
  const target = url.searchParams.get("target") as ServiceTarget | null;
  const path = url.searchParams.get("path");

  if (!target || !VALID_TARGETS.includes(target) || !path || !isAllowedPath(path)) {
    return Response.json(
      { error: "Invalid or missing target/path (path must match an allowed API prefix)" },
      { status: 400 }
    );
  }

  let cfg: Record<string, unknown>;
  try {
    cfg = await fetchGoClientConfig();
  } catch (e) {
    return Response.json(
      { error: e instanceof Error ? e.message : "Failed to load polyback client config" },
      { status: 503 }
    );
  }

  const base = baseForTarget(cfg, target);
  if (!base) {
    return Response.json(
      { error: `No serviceUrls.${target} in client config (is that process in develop.yaml?)` },
      { status: 502 }
    );
  }

  const upstream = `${base}${path}`;
  try {
    const upstreamRes = await fetch(upstream, {
      method: "GET",
      headers: { Accept: "application/json" },
    });
    const text = await upstreamRes.text();
    const ct = upstreamRes.headers.get("content-type") || "application/json";
    return new Response(text, {
      status: upstreamRes.status,
      headers: { "Content-Type": ct },
    });
  } catch (e) {
    return Response.json(
      { error: e instanceof Error ? e.message : "Relay fetch failed", upstream },
      { status: 502 }
    );
  }
}
