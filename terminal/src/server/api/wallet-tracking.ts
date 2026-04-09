import WebSocket from "ws";

const activeConnections = new Map<
  string,
  {
    ws: WebSocket;
    subscriptionId: string;
    apiKey: string;
  }
>();

/** Active WebSocket slots per Dome API key string (stable when the key list grows or reorders). */
const wsUsageByApiKey = new Map<string, number>();

function usageOf(apiKey: string): number {
  return wsUsageByApiKey.get(apiKey) ?? 0;
}

function incrementUsage(apiKey: string) {
  wsUsageByApiKey.set(apiKey, usageOf(apiKey) + 1);
}

function decrementUsage(apiKey: string) {
  const n = Math.max(0, usageOf(apiKey) - 1);
  if (n === 0) {
    wsUsageByApiKey.delete(apiKey);
  } else {
    wsUsageByApiKey.set(apiKey, n);
  }
}

function parseDomeApiKeys(): string[] {
  const keys: string[] = [];
  const multi = process.env.DOME_API_KEYS?.trim();
  if (multi) {
    for (const raw of multi.split(/[,\n\s]+/)) {
      const k = raw.trim();
      if (k) keys.push(k);
    }
  }
  const single = process.env.DOME_API_KEY?.trim();
  if (single && !keys.includes(single)) {
    keys.unshift(single);
  }
  if (keys.length === 0 && single) {
    keys.push(single);
  }
  return [...new Set(keys)];
}

function getDomeKeyRing(): { keys: string[]; maxPerKey: number } {
  const keys = parseDomeApiKeys();
  const rawMax = process.env.DOME_WS_MAX_SUBSCRIPTIONS_PER_KEY?.trim();
  const parsed = rawMax ? Number.parseInt(rawMax, 10) : NaN;
  const maxPerKey = Number.isFinite(parsed) && parsed > 0 ? parsed : 2;
  return { keys, maxPerKey };
}

/**
 * Picks the configured key with the fewest active streams under maxPerKey.
 * Total pool capacity is keys.length × maxPerKey (each new key adds maxPerKey slots automatically).
 */
function tryAcquireSlot(ring: { keys: string[]; maxPerKey: number }): { apiKey: string; keyOrdinal: number } | null {
  const { keys, maxPerKey } = ring;
  if (keys.length === 0) return null;

  let best: string | null = null;
  let bestIdx = -1;
  let bestUsage = Infinity;
  for (let i = 0; i < keys.length; i++) {
    const k = keys[i]!;
    const u = usageOf(k);
    if (u >= maxPerKey) continue;
    if (u < bestUsage) {
      bestUsage = u;
      best = k;
      bestIdx = i;
    }
  }
  if (!best || bestIdx < 0) return null;
  incrementUsage(best);
  return { apiKey: best, keyOrdinal: bestIdx };
}

function sseHeaders(): HeadersInit {
  return {
    "Content-Type": "text/event-stream",
    "Cache-Control": "no-cache",
    Connection: "keep-alive",
  };
}

/** Snapshot for /api/wallet-tracking/status (no secrets). */
export function getWalletTrackingBackendState() {
  const ring = getDomeKeyRing();
  const usageByKeyIndex = ring.keys.map((k) => usageOf(k));
  const slotsInUse = usageByKeyIndex.reduce((a, b) => a + b, 0);
  return {
    dome_configured: ring.keys.length > 0,
    dome_key_count: ring.keys.length,
    max_subscriptions_per_key: ring.maxPerKey,
    max_concurrent_streams: ring.keys.length * ring.maxPerKey,
    active_stream_connections: activeConnections.size,
    slots_in_use: slotsInUse,
    usage_by_key_index: usageByKeyIndex,
  };
}

export async function GET(request: Request) {
  const { searchParams } = new URL(request.url);
  const walletAddress = searchParams.get("wallet");

  if (!walletAddress) {
    return new Response(JSON.stringify({ error: "Wallet address is required" }), {
      status: 400,
      headers: { "Content-Type": "application/json" },
    });
  }

  if (!walletAddress.match(/^0x[a-fA-F0-9]{40}$/)) {
    return new Response(JSON.stringify({ error: "Invalid wallet address format" }), {
      status: 400,
      headers: { "Content-Type": "application/json" },
    });
  }

  const ring = getDomeKeyRing();
  if (ring.keys.length === 0) {
    return new Response(JSON.stringify({ error: "DOME_API_KEY (or DOME_API_KEYS) not configured" }), {
      status: 500,
      headers: { "Content-Type": "application/json" },
    });
  }

  const totalCap = ring.keys.length * ring.maxPerKey;
  const slot = tryAcquireSlot(ring);
  if (!slot) {
    const encoder = new TextEncoder();
    const capBody = new ReadableStream({
      start(controller) {
        const message = JSON.stringify({
          type: "error",
          data: {
            error: `All Dome API keys are at capacity (${ring.keys.length} key(s) × ${ring.maxPerKey} stream(s) each = ${totalCap} total). Add more keys to DOME_API_KEYS (capacity scales automatically) or raise DOME_WS_MAX_SUBSCRIPTIONS_PER_KEY if your Dome tier allows more subscriptions per key.`,
            code: "DOME_WS_CAP",
          },
          timestamp: new Date().toISOString(),
        });
        controller.enqueue(encoder.encode(`data: ${message}\n\n`));
        controller.close();
      },
    });
    return new Response(capBody, { headers: sseHeaders() });
  }

  const { apiKey, keyOrdinal } = slot;
  const encoder = new TextEncoder();
  let isConnectionClosed = false;
  let heartbeatInterval: ReturnType<typeof setInterval> | null = null;
  let connectionId: string | null = null;
  let ws: WebSocket | null = null;
  let subscriptionId: string | null = null;
  let cleanedUp = false;

  const cleanup = () => {
    if (cleanedUp) return;
    cleanedUp = true;
    isConnectionClosed = true;
    decrementUsage(apiKey);
    if (heartbeatInterval) {
      clearInterval(heartbeatInterval);
      heartbeatInterval = null;
    }
    if (connectionId) {
      activeConnections.delete(connectionId);
      connectionId = null;
    }
    if (ws) {
      try {
        ws.close();
      } catch {
        /* ignore */
      }
      ws = null;
    }
  };

  const stream = new ReadableStream({
    start(controller) {
      if (request.signal.aborted) {
        cleanup();
        try {
          controller.close();
        } catch {
          /* ignore */
        }
        return;
      }

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

      request.signal.addEventListener("abort", () => {
        console.log("[Dome WS] Client disconnected, cleaning up...");
        cleanup();
      });

      try {
        const wsUrl = `wss://ws.domeapi.io/${apiKey}`;
        console.log(
          "[Dome WS] Connecting (key",
          `${keyOrdinal + 1}/${ring.keys.length}`,
          "):",
          wsUrl.replace(apiKey, "***")
        );

        ws = new WebSocket(wsUrl);

        ws.on("open", () => {
          console.log("[Dome WS] Connection opened (key", `${keyOrdinal + 1}/${ring.keys.length}`, ")");
          sendEvent("connected", { message: "WebSocket connected to Dome" });

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

              connectionId = `${walletAddress}-${Date.now()}`;
              activeConnections.set(connectionId, {
                ws: ws!,
                subscriptionId: subscriptionId!,
                apiKey,
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
          cleanup();
        });

        ws.on("error", (error) => {
          console.error("[Dome WS] Error:", error.message);
          sendEvent("error", { error: error.message || "WebSocket error" });
        });

        heartbeatInterval = setInterval(() => {
          if (!isConnectionClosed && ws?.readyState === WebSocket.OPEN) {
            sendEvent("heartbeat", { status: "alive" });
          }
        }, 30000);
      } catch (error) {
        console.error("[Dome WS] Setup error:", error);
        const errorMessage = error instanceof Error ? error.message : "Failed to connect to Dome WebSocket";
        sendEvent("error", { error: errorMessage });
        cleanup();
        try {
          controller.close();
        } catch {
          /* ignore */
        }
      }
    },
  });

  return new Response(stream, {
    headers: sseHeaders(),
  });
}
