import { listAgentRuns } from "@/server/local-run-log-db";

/**
 * GET /api/agent-runs?feature=arbitrage_finder&limit=50
 * Returns recent rows from the local SQLite run log (no secrets).
 */
export async function GET(request: Request): Promise<Response> {
  const url = new URL(request.url);
  const feature = url.searchParams.get("feature")?.trim() || null;
  const limitRaw = url.searchParams.get("limit");
  const limit = limitRaw ? parseInt(limitRaw, 10) : 50;
  const limitSafe = Number.isFinite(limit) ? limit : 50;

  const rows = listAgentRuns({ feature, limit: limitSafe });

  return Response.json({
    success: true,
    rows: rows.map((r) => ({
      id: r.id,
      createdAt: r.created_at,
      feature: r.feature,
      success: r.success === 1,
      httpStatus: r.http_status,
      errorMessage: r.error_message,
      model: r.model,
      processingTimeMs: r.processing_time_ms,
      requestSummary: r.request_summary,
      responseSummary: r.response_summary,
    })),
  });
}
