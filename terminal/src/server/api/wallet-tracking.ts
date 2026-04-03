import WebSocket from "ws";

// Store active connections
const activeConnections = new Map<string, {
  ws: WebSocket;
  subscriptionId: string;
}>();

export async function GET(request: Request) {
  const { searchParams } = new URL(request.url);
  const walletAddress = searchParams.get("wallet");

  if (!walletAddress) {
    return new Response(JSON.stringify({ error: "Wallet address is required" }), {
      status: 400,
      headers: { "Content-Type": "application/json" },
    });
  }

  // Validate wallet address format
  if (!walletAddress.match(/^0x[a-fA-F0-9]{40}$/)) {
    return new Response(JSON.stringify({ error: "Invalid wallet address format" }), {
      status: 400,
      headers: { "Content-Type": "application/json" },
    });
  }

  const apiKey = process.env.DOME_API_KEY;
  if (!apiKey) {
    return new Response(JSON.stringify({ error: "DOME_API_KEY not configured" }), {
      status: 500,
      headers: { "Content-Type": "application/json" },
    });
  }

  // Create a streaming response using SSE
  const encoder = new TextEncoder();
  let isConnectionClosed = false;
  let heartbeatInterval: ReturnType<typeof setInterval> | null = null;
  let connectionId: string | null = null;
  let ws: WebSocket | null = null;
  let subscriptionId: string | null = null;

  const stream = new ReadableStream({
    async start(controller) {
      const sendEvent = (type: string, data: unknown) => {
        if (isConnectionClosed) return;
        try {
          const message = JSON.stringify({
            type,
            data,
            timestamp: new Date().toISOString(),
          });
          controller.enqueue(encoder.encode(`data: ${message}\n\n`));
        } catch {
          // Connection might be closed
        }
      };

      try {
        // Direct WebSocket connection to Dome (bypassing SDK for debugging)
        const wsUrl = `wss://ws.domeapi.io/${apiKey}`;
        console.log("[Dome WS] Connecting to:", wsUrl.replace(apiKey, "***"));
        
        ws = new WebSocket(wsUrl);

        ws.on("open", () => {
          console.log("[Dome WS] Connection opened");
          sendEvent("connected", { message: "WebSocket connected to Dome" });

          // Send subscription message
          const subscribeMessage = {
            action: "subscribe",
            platform: "polymarket",
            version: 1,
            type: "orders",
            filters: {
              users: [walletAddress.toLowerCase()],
            },
          };
          
          console.log("[Dome WS] Sending subscription:", JSON.stringify(subscribeMessage));
          ws!.send(JSON.stringify(subscribeMessage));
        });

        ws.on("message", (data) => {
          try {
            const message = JSON.parse(data.toString());
            console.log("[Dome WS] Message received:", JSON.stringify(message));

            if (message.type === "ack") {
              subscriptionId = message.subscription_id;
              console.log("[Dome WS] Subscription acknowledged:", subscriptionId);
              sendEvent("subscribed", {
                subscription_id: subscriptionId,
                message: `Subscribed to wallet: ${walletAddress}`,
              });

              // Store the connection
              connectionId = `${walletAddress}-${Date.now()}`;
              activeConnections.set(connectionId, {
                ws: ws!,
                subscriptionId: subscriptionId!,
              });
            } else if (message.type === "event" && message.data) {
              const order = message.data;
              console.log("[Dome WS] Order received:", order.market_slug);
              sendEvent("order", {
                token_id: order.token_id,
                token_label: order.token_label,
                side: order.side,
                market_slug: order.market_slug,
                condition_id: order.condition_id,
                shares: order.shares,
                shares_normalized: order.shares_normalized,
                price: order.price,
                tx_hash: order.tx_hash,
                title: order.title,
                timestamp: order.timestamp,
                order_hash: order.order_hash,
                user: order.user,
                taker: order.taker,
              });
            }
          } catch (e) {
            console.error("[Dome WS] Failed to parse message:", e);
          }
        });

        ws.on("close", (code, reason) => {
          console.log("[Dome WS] Connection closed. Code:", code, "Reason:", reason?.toString() || "none");
          if (!isConnectionClosed) {
            sendEvent("disconnected", { 
              message: "WebSocket disconnected",
              code,
              reason: reason?.toString() || "unknown",
            });
          }
        });

        ws.on("error", (error) => {
          console.error("[Dome WS] Error:", error.message);
          sendEvent("error", { error: error.message || "WebSocket error" });
        });

        // Send periodic heartbeats (every 30 seconds)
        heartbeatInterval = setInterval(() => {
          if (!isConnectionClosed && ws?.readyState === WebSocket.OPEN) {
            sendEvent("heartbeat", { status: "alive" });
          }
        }, 30000);

        // Handle stream cancellation
        request.signal.addEventListener("abort", () => {
          console.log("[Dome WS] Client disconnected, cleaning up...");
          isConnectionClosed = true;
          if (heartbeatInterval) {
            clearInterval(heartbeatInterval);
          }
          if (ws) {
            ws.close();
          }
          if (connectionId) {
            activeConnections.delete(connectionId);
          }
        });

      } catch (error) {
        console.error("[Dome WS] Setup error:", error);
        const errorMessage = error instanceof Error ? error.message : "Failed to connect to Dome WebSocket";
        sendEvent("error", { error: errorMessage });
        
        // Clean up on error
        if (ws) {
          ws.close();
        }
        if (heartbeatInterval) {
          clearInterval(heartbeatInterval);
        }
        
        controller.close();
      }
    },
  });

  return new Response(stream, {
    headers: {
      "Content-Type": "text/event-stream",
      "Cache-Control": "no-cache",
      "Connection": "keep-alive",
    },
  });
}

