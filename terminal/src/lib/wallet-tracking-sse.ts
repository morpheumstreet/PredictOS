import type { OrderEvent, SSEMessage, WalletTrackingLogEntry } from "@/types/wallet-tracking";

export function formatOrderMessage(order: OrderEvent): string {
  const side = order.side === "BUY" ? "📈 BUY" : "📉 SELL";
  const price = (order.price * 100).toFixed(1);
  const shares = order.shares_normalized?.toFixed(2) || order.shares;
  return `${side} ${shares} shares @ ${price}¢ — ${order.title || order.market_slug}`;
}

function ts(message: SSEMessage): string {
  return message.timestamp || new Date().toISOString();
}

/**
 * Maps one SSE payload to a log line, or null to skip (heartbeat / unknown).
 */
export function sseMessageToLogEntry(message: SSEMessage): WalletTrackingLogEntry | null {
  switch (message.type) {
    case "connected":
      return { timestamp: ts(message), level: "SUCCESS", message: "Connected to Dome WebSocket" };
    case "subscribed": {
      const subData = message.data as { subscription_id?: string; message?: string };
      return {
        timestamp: ts(message),
        level: "SUCCESS",
        message: subData.message || "Subscribed to wallet",
      };
    }
    case "order": {
      const order = message.data as OrderEvent;
      return {
        timestamp: ts(message),
        level: "ORDER",
        message: formatOrderMessage(order),
        details: {
          user: order.user,
          tx_hash: order.tx_hash,
          market_slug: order.market_slug,
          price: order.price,
          shares: order.shares_normalized,
        },
      };
    }
    case "error": {
      const errData = message.data as { error?: string };
      return {
        timestamp: ts(message),
        level: "ERROR",
        message: errData.error || "Unknown error occurred",
      };
    }
    case "disconnected":
      return {
        timestamp: ts(message),
        level: "WARN",
        message: "WebSocket disconnected, attempting to reconnect...",
      };
    case "heartbeat":
      return null;
    default:
      return null;
  }
}

export type WalletTrackingSSECallbacks = {
  onLog: (entry: WalletTrackingLogEntry) => void;
  /** Called when EventSource ends (closed / error). */
  onEnded: () => void;
};

/**
 * Opens SSE for one wallet. Caller must call `eventSource.close()` on stop/unmount.
 */
export function openWalletTrackingEventSource(
  walletAddress: string,
  { onLog, onEnded }: WalletTrackingSSECallbacks
): EventSource {
  const eventSource = new EventSource(
    `/api/wallet-tracking?wallet=${encodeURIComponent(walletAddress)}`
  );

  eventSource.onmessage = (event) => {
    try {
      const message = JSON.parse(event.data) as SSEMessage;
      const entry = sseMessageToLogEntry(message);
      if (entry) onLog(entry);
    } catch (e) {
      console.error("Failed to parse SSE message:", e);
    }
  };

  eventSource.onerror = () => {
    if (eventSource.readyState === EventSource.CLOSED) {
      onEnded();
    }
  };

  return eventSource;
}
