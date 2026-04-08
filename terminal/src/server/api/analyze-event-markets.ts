import { intelligenceApiUrl } from "@/lib/intelligence-url";

/**
 * Proxies to Polyback Intelligence analyze-event-markets (get-events + LLM in one call).
 */
export async function POST(request: Request) {
  try {
    const body = await request.json();
    const url =
      process.env.INTELLIGENCE_EDGE_FUNCTION_ANALYZE_EVENT_MARKETS?.trim() ||
      intelligenceApiUrl("analyze-event-markets");
    const res = await fetch(url, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    const data = await res.json();
    return Response.json(data, { status: res.status });
  } catch (e) {
    return Response.json(
      { success: false, error: e instanceof Error ? e.message : "error" },
      { status: 500 }
    );
  }
}
