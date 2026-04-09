/**
 * Types for the Wallet Tracking feature using Dome API
 */

/**
 * Log entry for wallet tracking events
 */
export interface WalletTrackingLogEntry {
  timestamp: string;
  level: "INFO" | "WARN" | "ERROR" | "SUCCESS" | "ORDER";
  message: string;
  details?: Record<string, unknown>;
}

/**
 * Order event from Dome WebSocket
 */
export interface OrderEvent {
  token_id: string;
  side: "BUY" | "SELL";
  market_slug: string;
  condition_id: string;
  shares: string;
  shares_normalized: number;
  price: number;
  tx_hash: string;
  title: string;
  timestamp: number;
  order_hash: string;
  user: string;
}

/**
 * SSE event types
 */
export type SSEEventType = 
  | "connected"
  | "subscribed"
  | "order"
  | "error"
  | "heartbeat"
  | "disconnected";

/**
 * SSE message payload
 */
export interface SSEMessage {
  type: SSEEventType;
  data?: OrderEvent | { subscription_id?: string; message?: string; error?: string };
  timestamp: string;
}

/** sessionStorage key for multi-bot wallet tracking state (addresses + selection + running). */
export const WALLET_TRACKING_STORAGE_KEY_V2 = "predictos-wallet-tracking-v2";

/** Legacy single-address key; migrated into v2 on first read. */
export const WALLET_TRACKING_STORAGE_KEY_V1 = "predictos-wallet-tracking-address";

export interface WalletTrackingPersistedV2 {
  /** Normalized addresses (lowercase), UI order. */
  bots: string[];
  selected: string | null;
  /** Subset of `bots` that were streaming last session (auto-resume). */
  running: string[];
}

/** In-memory bot row (keyed by normalized address). */
export interface WalletTrackerBot {
  /** Canonical `0x` + 40 hex lowercase */
  addressKey: string;
  isRunning: boolean;
  logs: WalletTrackingLogEntry[];
}

export const WALLET_TRACKING_LOG_CAP = 500;

const WALLET_REGEX = /^0x[a-fA-F0-9]{40}$/;

export function isValidWalletAddress(raw: string): boolean {
  return WALLET_REGEX.test(raw.trim());
}

/** Returns normalized key or null if invalid. */
export function normalizeWalletAddress(raw: string): string | null {
  const t = raw.trim().toLowerCase();
  return WALLET_REGEX.test(t) ? t : null;
}

export function shortWalletLabel(addressKey: string): string {
  if (addressKey.length < 10) return addressKey;
  return `${addressKey.slice(0, 6)}…${addressKey.slice(-4)}`;
}

export type WalletTrackerBackendStatus = "checking" | "ready" | "misconfigured" | "error";
