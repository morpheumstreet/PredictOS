import type { EventAnalysisAgentRequest, EventAnalysisAgentResponse } from "@/types/agentic";
import { intelligenceApiUrl } from "@/lib/intelligence-url";

/**
 * Server-side API route to proxy requests to Polyback Intelligence (event-analysis-agent).
 */
export async function POST(request: Request) {
  try {
    const body: EventAnalysisAgentRequest = await request.json();

    // Validate required fields
    if (!body.markets || !Array.isArray(body.markets) || body.markets.length === 0) {
      return Response.json(
        {
          success: false,
          error: "Missing required field: markets",
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
      process.env.INTELLIGENCE_EDGE_FUNCTION_EVENT_ANALYSIS_AGENT?.trim() ||
      intelligenceApiUrl("event-analysis-agent");

    const response = await fetch(url, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        markets: body.markets,
        eventIdentifier: body.eventIdentifier,
        pmType: body.pmType,
        model: body.model,
        question: body.question,
        tools: body.tools,
        userCommand: body.userCommand,
      }),
    });

    const data: EventAnalysisAgentResponse = await response.json();

    return Response.json(data, { status: response.status });
  } catch (error) {
    console.error("Error in event-analysis-agent API route:", error);
    return Response.json(
      {
        success: false,
        error: error instanceof Error ? error.message : "An unexpected error occurred",
      },
      { status: 500 }
    );
  }
}

