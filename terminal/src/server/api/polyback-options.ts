/**
 * GET /api/polyback/options — tells the browser whether to call polyback-mm directly.
 * When POLYBACK_BROWSER_BOOTSTRAP_URL is set (e.g. http://127.0.0.1:8080), enable CORS
 * on polyback-mm (server.cors_allowed_origins in YAML) to match the terminal origin.
 */
export async function GET(): Promise<Response> {
  const v = process.env.POLYBACK_BROWSER_BOOTSTRAP_URL?.trim();
  const browserDirectBootstrap = v && v.length > 0 ? v.replace(/\/$/, "") : null;
  return Response.json({ browserDirectBootstrap });
}
