import type { PolyfactualResearchRequest, PolyfactualResearchResponse } from "@/types/polyfactual";
import { intelligenceApiUrl } from "@/lib/intelligence-url";

/**
 * Server-side API route to proxy requests to Polyback Intelligence (polyfactual-research).
 * Polyfactual API key is configured on the intelligence process (POLYFACTUAL_API_KEY).
 */
export async function POST(request: Request) {
  try {
    // Parse request body
    const body: PolyfactualResearchRequest = await request.json();

    // Validate required fields
    if (!body.query || typeof body.query !== "string") {
      return Response.json(
        {
          success: false,
          error: "Missing required field: query",
        },
        { status: 400 }
      );
    }

    const trimmedQuery = body.query.trim();
    if (!trimmedQuery) {
      return Response.json(
        {
          success: false,
          error: "Query cannot be empty",
        },
        { status: 400 }
      );
    }

    // Validate query length (Polyfactual limit is 1000 characters)
    if (trimmedQuery.length > 1000) {
      return Response.json(
        {
          success: false,
          error: "Query exceeds maximum length of 1000 characters",
        },
        { status: 400 }
      );
    }

    const url =
      process.env.INTELLIGENCE_EDGE_FUNCTION_POLYFACTUAL_RESEARCH?.trim() ||
      intelligenceApiUrl("polyfactual-research");

    const response = await fetch(url, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        query: trimmedQuery,
      }),
    });

    const data: PolyfactualResearchResponse = await response.json();

    return Response.json(data, { status: response.status });
  } catch (error) {
    console.error("Error in polyfactual-research API route:", error);
    return Response.json(
      {
        success: false,
        error: error instanceof Error ? error.message : "An unexpected error occurred",
      },
      { status: 500 }
    );
  }
}

