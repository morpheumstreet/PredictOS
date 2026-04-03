/**
 * API Route: x402-seller
 * 
 * Proxies requests to the x402-seller Supabase edge function.
 * Supports listing bazaar sellers and calling x402-protected endpoints.
 */


const SUPABASE_URL = process.env.SUPABASE_URL || "http://127.0.0.1:54321";
const SUPABASE_ANON_KEY = process.env.SUPABASE_ANON_KEY || "";

export async function POST(request: Request) {
  try {
    const body = await request.json();
    
    console.log("[x402-seller API] Received request:", body.action);

    const response = await fetch(
      `${SUPABASE_URL}/functions/v1/x402-seller`,
      {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Authorization": `Bearer ${SUPABASE_ANON_KEY}`,
        },
        body: JSON.stringify(body),
      }
    );

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

