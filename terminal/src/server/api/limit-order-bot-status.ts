import { intelligenceApiUrl } from "@/lib/intelligence-url";

export type LimitOrderBotStatusResponse = {
  success: boolean;
  parallelLimitOrderPlacements?: number;
  description?: string;
  error?: string;
  intelligenceUrl?: string;
};

/**
 * GET /api/limit-order-bot/status — proxy to Polyback Intelligence (or edge) status for 15m bot parallelism.
 */
export async function GET(): Promise<Response> {
  const postBase =
    process.env.INTELLIGENCE_EDGE_FUNCTION_LIMIT_ORDER_BOT?.trim() ||
    intelligenceApiUrl("polymarket-up-down-15-markets-limit-order-bot");
  const statusUrl = `${postBase.replace(/\/$/, "")}/status`;

  try {
    const res = await fetch(statusUrl, {
      method: "GET",
      headers: { Accept: "application/json" },
      signal: AbortSignal.timeout(8000),
    });

    const text = await res.text();
    let body: unknown;
    try {
      body = text ? JSON.parse(text) : null;
    } catch {
      return Response.json(
        {
          success: false,
          error: "Backend returned non-JSON",
          intelligenceUrl: statusUrl,
        } satisfies LimitOrderBotStatusResponse,
        { status: 502 }
      );
    }

    const o = body as Record<string, unknown>;
    const parallel =
      typeof o.parallelLimitOrderPlacements === "number" && Number.isFinite(o.parallelLimitOrderPlacements)
        ? Math.floor(o.parallelLimitOrderPlacements)
        : undefined;

    if (!res.ok) {
      return Response.json(
        {
          success: false,
          error: typeof o.error === "string" ? o.error : `HTTP ${res.status}`,
          intelligenceUrl: statusUrl,
          parallelLimitOrderPlacements: parallel,
        } satisfies LimitOrderBotStatusResponse,
        { status: res.status >= 400 && res.status < 600 ? res.status : 502 }
      );
    }

    return Response.json({
      success: Boolean(o.success !== false),
      parallelLimitOrderPlacements: parallel,
      description: typeof o.description === "string" ? o.description : undefined,
      intelligenceUrl: statusUrl,
    } satisfies LimitOrderBotStatusResponse);
  } catch (e) {
    const msg = e instanceof Error ? e.message : "Request failed";
    return Response.json(
      {
        success: false,
        error: msg,
        intelligenceUrl: statusUrl,
      } satisfies LimitOrderBotStatusResponse,
      { status: 503 }
    );
  }
}
