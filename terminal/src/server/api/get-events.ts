import type { GetEventsRequest, GetEventsResponse } from "@/types/agentic";
import { intelligenceApiUrl } from "@/lib/intelligence-url";

/**
 * Server-side API route to proxy requests to Polyback Intelligence (get-events).
 */
export async function POST(request: Request) {
  try {
    const body: GetEventsRequest = await request.json();

    if (!body.url) {
      return Response.json(
        {
          success: false,
          error: "Missing required field: url",
        },
        { status: 400 }
      );
    }

    const lowerUrl = body.url.toLowerCase();
    const isKalshiBased = lowerUrl.includes("kalshi") || lowerUrl.includes("jup.ag/prediction");
    const dataProvider = isKalshiBased ? "dflow" : "dome";

    const url =
      process.env.INTELLIGENCE_EDGE_FUNCTION_GET_EVENTS?.trim() ||
      intelligenceApiUrl("get-events");

    const response = await fetch(url, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        url: body.url,
        dataProvider,
      }),
    });

    const data: GetEventsResponse = await response.json();

    return Response.json(data, { status: response.status });
  } catch (error) {
    console.error("Error in get-events API route:", error);
    return Response.json(
      {
        success: false,
        error: error instanceof Error ? error.message : "An unexpected error occurred",
      },
      { status: 500 }
    );
  }
}
