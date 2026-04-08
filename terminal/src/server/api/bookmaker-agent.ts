import type { AnalysisAggregatorRequest, AnalysisAggregatorResponse } from "@/types/agentic";
import { intelligenceApiUrl } from "@/lib/intelligence-url";

/**
 * Server-side API route to proxy requests to Polyback Intelligence (bookmaker-agent).
 */
export async function POST(request: Request) {
  try {
    const body: AnalysisAggregatorRequest = await request.json();

    // Validate required fields - need at least 2 total data sources (analyses + x402Results)
    const analysesCount = body.analyses?.length || 0;
    const x402Count = body.x402Results?.length || 0;
    const totalSources = analysesCount + x402Count;
    
    if (totalSources < 2) {
      return Response.json(
        {
          success: false,
          error: `Need at least 2 data sources to aggregate (got ${analysesCount} analyses + ${x402Count} PayAI results)`,
        },
        { status: 400 }
      );
    }

    if (!body.eventIdentifier) {
      return Response.json(
        {
          success: false,
          error: "Missing required field: eventIdentifier",
        },
        { status: 400 }
      );
    }

    if (!body.pmType) {
      return Response.json(
        {
          success: false,
          error: "Missing required field: pmType",
        },
        { status: 400 }
      );
    }

    if (!body.model) {
      return Response.json(
        {
          success: false,
          error: "Missing required field: model",
        },
        { status: 400 }
      );
    }

    const url =
      process.env.INTELLIGENCE_EDGE_FUNCTION_BOOKMAKER_AGENT?.trim() ||
      intelligenceApiUrl("bookmaker-agent");

    const response = await fetch(url, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        analyses: body.analyses || [],
        x402Results: body.x402Results || [],
        eventIdentifier: body.eventIdentifier,
        pmType: body.pmType,
        model: body.model,
      }),
    });

    const data: AnalysisAggregatorResponse = await response.json();

    return Response.json(data, { status: response.status });
  } catch (error) {
    console.error("Error in bookmaker-agent API route:", error);
    return Response.json(
      {
        success: false,
        error: error instanceof Error ? error.message : "An unexpected error occurred",
      },
      { status: 500 }
    );
  }
}

