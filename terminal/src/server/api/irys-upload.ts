import {
  getIrysUploadServiceBaseURL,
  validateIrysUploadProxyConfig,
} from "@/server/lib/irys-config";

/**
 * POST /api/irys-upload — proxies to the Go irys-upload service (Solana + Irys).
 */
export async function POST(request: Request): Promise<Response> {
  const configValidation = validateIrysUploadProxyConfig();
  if (!configValidation.valid) {
    console.error("[Irys Upload] Configuration error:", configValidation.error);
    return Response.json(
      {
        success: false,
        error: `Irys configuration error: ${configValidation.error}`,
      },
      { status: 500 }
    );
  }

  const base = getIrysUploadServiceBaseURL()!;
  let payload: {
    requestId?: string;
    agentsData?: unknown[];
  };
  try {
    payload = await request.json();
  } catch {
    return Response.json(
      { success: false, error: "Invalid JSON body" },
      { status: 400 }
    );
  }

  if (!payload.requestId) {
    return Response.json(
      { success: false, error: "Missing required field: requestId" },
      { status: 400 }
    );
  }
  if (!payload.agentsData || payload.agentsData.length === 0) {
    return Response.json(
      { success: false, error: "Missing required field: agentsData" },
      { status: 400 }
    );
  }

  const bodyText = JSON.stringify(payload);

  let upstream: Response;
  try {
    upstream = await fetch(`${base}/upload`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: bodyText,
    });
  } catch (e) {
    const msg = e instanceof Error ? e.message : "Unknown network error";
    console.error("[Irys Upload] Upstream unreachable:", msg);
    return Response.json(
      {
        success: false,
        error: `Irys upload service unreachable (${msg}). Is mm/irys-upload running?`,
      },
      { status: 503 }
    );
  }

  const text = await upstream.text();
  return new Response(text, {
    status: upstream.status,
    headers: { "Content-Type": "application/json" },
  });
}

/**
 * GET /api/irys-upload — proxies status from the Go service.
 */
export async function GET(_request: Request): Promise<Response> {
  const configValidation = validateIrysUploadProxyConfig();
  if (!configValidation.valid) {
    return Response.json({
      configured: false,
      environment: "not set",
      error: configValidation.error,
    });
  }

  const base = getIrysUploadServiceBaseURL()!;
  try {
    const upstream = await fetch(`${base}/status`);
    const text = await upstream.text();
    return new Response(text, {
      status: upstream.status,
      headers: { "Content-Type": "application/json" },
    });
  } catch (e) {
    const msg = e instanceof Error ? e.message : "Unknown network error";
    return Response.json({
      configured: false,
      environment: "not set",
      error: `Irys upload service unreachable: ${msg}`,
    });
  }
}
