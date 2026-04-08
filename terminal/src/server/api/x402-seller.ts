import { intelligenceApiUrl } from "@/lib/intelligence-url";

/**
 * API Route: x402-seller — proxies to Polyback Intelligence.
 */
export async function POST(request: Request) {
  try {
    const body = await request.json();

    const url =
      process.env.INTELLIGENCE_EDGE_FUNCTION_X402_SELLER?.trim() ||
      intelligenceApiUrl("x402-seller");

    const response = await fetch(url, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(body),
    });

    const data = await response.json();

    return Response.json(data, { status: response.status });
  } catch (error) {
    console.error("[x402-seller API] Error:", error);
    return Response.json(
      {
        success: false,
        error: error instanceof Error ? error.message : "Internal server error",
      },
      { status: 500 }
    );
  }
}

export async function OPTIONS(_request: Request): Promise<Response> {
  return new Response(null, {
    status: 200,
    headers: {
      "Access-Control-Allow-Origin": "*",
      "Access-Control-Allow-Methods": "POST, OPTIONS",
      "Access-Control-Allow-Headers": "Content-Type, Authorization",
    },
  });
}
