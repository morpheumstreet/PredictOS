import type { ArbitrageRequest, ArbitrageResponse } from "@/types/arbitrage";
import { tryInsertAgentRun } from "@/server/local-run-log-db";

const FEATURE = "arbitrage_finder";

function compactArbitrageResponseSummary(data: ArbitrageResponse): string {
  if (!data.success) {
    return JSON.stringify({
      success: false,
      error: data.error ?? null,
    });
  }
  const d = data.data;
  if (!d) {
    return JSON.stringify({ success: true, data: null });
  }
  return JSON.stringify({
    isSameMarket: d.isSameMarket,
    sameMarketConfidence: d.sameMarketConfidence,
    hasArbitrage: d.arbitrage?.hasArbitrage ?? false,
    profitPercent: d.arbitrage?.profitPercent ?? null,
    viableAfterFees: d.arbitrage?.feeAdjusted?.viableAfterFees ?? null,
    sourceMarket: data.metadata?.sourceMarket,
    searchedMarket: data.metadata?.searchedMarket,
  });
}

/**
 * Server-side API route to proxy requests to the arbitrage-finder Edge Function.
 */
export async function POST(request: Request) {
  const runId = crypto.randomUUID();
  const createdAtMs = Date.now();

  const logRequest = (params: {
    body: ArbitrageRequest | null;
    success: boolean;
    httpStatus: number | null;
    errorMessage: string | null;
    model: string | null;
    processingTimeMs: number | null;
    requestSummary: string;
    responseSummary: string;
  }) => {
    tryInsertAgentRun({
      id: runId,
      createdAtMs,
      feature: FEATURE,
      success: params.success,
      httpStatus: params.httpStatus,
      errorMessage: params.errorMessage,
      model: params.model,
      processingTimeMs: params.processingTimeMs,
      requestSummary: params.requestSummary,
      responseSummary: params.responseSummary,
    });
  };

  try {
    const supabaseUrl = process.env.SUPABASE_URL;
    const supabaseAnonKey = process.env.SUPABASE_ANON_KEY;

    let body: ArbitrageRequest;
    try {
      body = await request.json();
    } catch {
      logRequest({
        body: null,
        success: false,
        httpStatus: 400,
        errorMessage: "Invalid JSON body",
        model: null,
        processingTimeMs: null,
        requestSummary: "{}",
        responseSummary: JSON.stringify({ parseError: "request_json" }),
      });
      return Response.json(
        {
          success: false,
          error: "Invalid JSON in request body",
        },
        { status: 400 }
      );
    }

    if (!supabaseUrl || !supabaseAnonKey) {
      const reqSummary = JSON.stringify({ url: body.url ?? "", model: body.model ?? "" });
      logRequest({
        body,
        success: false,
        httpStatus: 500,
        errorMessage: "Missing Supabase credentials",
        model: body.model ?? null,
        processingTimeMs: null,
        requestSummary: reqSummary,
        responseSummary: JSON.stringify({ configError: "supabase" }),
      });
      return Response.json(
        {
          success: false,
          error: "Server configuration error: Missing Supabase credentials",
        },
        { status: 500 }
      );
    }

    if (!body.url) {
      logRequest({
        body,
        success: false,
        httpStatus: 400,
        errorMessage: "Missing required field: url",
        model: body.model ?? null,
        processingTimeMs: null,
        requestSummary: JSON.stringify({ url: "", model: body.model ?? "" }),
        responseSummary: JSON.stringify({ validation: "url" }),
      });
      return Response.json(
        {
          success: false,
          error: "Missing required field: url",
        },
        { status: 400 }
      );
    }

    if (!body.model) {
      logRequest({
        body,
        success: false,
        httpStatus: 400,
        errorMessage: "Missing required field: model",
        model: null,
        processingTimeMs: null,
        requestSummary: JSON.stringify({ url: body.url, model: "" }),
        responseSummary: JSON.stringify({ validation: "model" }),
      });
      return Response.json(
        {
          success: false,
          error: "Missing required field: model",
        },
        { status: 400 }
      );
    }

    const requestSummary = JSON.stringify({ url: body.url, model: body.model });

    const edgeFunctionUrl =
      process.env.SUPABASE_EDGE_FUNCTION_ARBITRAGE_FINDER ||
      `${supabaseUrl}/functions/v1/arbitrage-finder`;

    const response = await fetch(edgeFunctionUrl, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${supabaseAnonKey}`,
        apikey: supabaseAnonKey,
      },
      body: JSON.stringify({
        url: body.url,
        model: body.model,
      }),
    });

    const rawText = await response.text();
    let data: ArbitrageResponse;
    try {
      data = JSON.parse(rawText) as ArbitrageResponse;
    } catch {
      logRequest({
        body,
        success: false,
        httpStatus: response.status,
        errorMessage: "Invalid JSON from arbitrage-finder service",
        model: body.model,
        processingTimeMs: null,
        requestSummary,
        responseSummary: JSON.stringify({
          parseError: "edge_response",
          snippet: rawText.slice(0, 200),
        }),
      });
      return Response.json(
        {
          success: false,
          error: "Invalid JSON from arbitrage-finder service",
        },
        { status: 502 }
      );
    }

    const processingTimeMs = data.metadata?.processingTimeMs ?? null;

    logRequest({
      body,
      success: !!data.success,
      httpStatus: response.status,
      errorMessage: data.error ?? null,
      model: body.model,
      processingTimeMs,
      requestSummary,
      responseSummary: compactArbitrageResponseSummary(data),
    });

    return Response.json(data, { status: response.status });
  } catch (error) {
    console.error("Error in arbitrage-finder API route:", error);
    const message = error instanceof Error ? error.message : "An unexpected error occurred";
    tryInsertAgentRun({
      id: runId,
      createdAtMs,
      feature: FEATURE,
      success: false,
      httpStatus: 500,
      errorMessage: message,
      model: null,
      processingTimeMs: null,
      requestSummary: "{}",
      responseSummary: JSON.stringify({ exception: "handler" }),
    });
    return Response.json(
      {
        success: false,
        error: message,
      },
      { status: 500 }
    );
  }
}
