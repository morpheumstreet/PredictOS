/**
 * Proxies GET /api/v1/config/client from polyback-mm (Go) for same-origin browser access.
 * Bootstrap URL: POLYBACK_BOOTSTRAP_URL (default http://127.0.0.1:8080).
 */
export async function GET(_request: Request): Promise<Response> {
  const bootstrap = process.env.POLYBACK_BOOTSTRAP_URL?.trim() || "http://127.0.0.1:8080";
  const base = bootstrap.replace(/\/$/, "");
  const upstream = `${base}/api/v1/config/client`;

  try {
    const res = await fetch(upstream, {
      headers: { Accept: "application/json" },
    });
    const text = await res.text();
    let parsed: unknown;
    try {
      parsed = JSON.parse(text);
    } catch {
      return Response.json(
        {
          success: false,
          error: "Invalid JSON from polyback config endpoint",
          upstream,
        },
        { status: 502 }
      );
    }

    const merged =
      typeof parsed === "object" && parsed !== null && !Array.isArray(parsed)
        ? { success: res.ok, ...(parsed as Record<string, unknown>) }
        : { success: res.ok, data: parsed };

    return Response.json(merged, { status: res.ok ? 200 : res.status });
  } catch (e) {
    return Response.json(
      {
        success: false,
        error: e instanceof Error ? e.message : "fetch failed",
        upstream,
      },
      { status: 503 }
    );
  }
}
