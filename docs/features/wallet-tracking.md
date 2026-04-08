# Wallet Tracking Setup

This document explains the architecture and configuration for the **Wallet Tracking** feature in PredictOS, which is at least 10 seconds faster than leading hosted bots in the industry. Delays of the order of milliseconds can cause huge losses. Developers should have access to the fastest trackers out there. 

Open-source copytrading will be released soon.

## Overview

The Wallet Tracking feature allows you to monitor real-time order activity on any Polymarket wallet. Enter any wallet address and watch trades flow in live as they happen — perfect for tracking whales, researching trader strategies, or monitoring your own positions.

> 🎯 **Key Differentiator:** This feature runs entirely in the frontend using Dome SDK's WebSocket API — no Supabase Edge Functions required.

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              WALLET TRACKING                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────┐      SSE Stream       ┌─────────────────────────────────┐  │
│  │   Browser   │ ◄────────────────────►│   Next.js API Route             │  │
│  │  Component  │                       │   /api/wallet-tracking          │  │
│  │             │                       │                                 │  │
│  │  - Input    │  EventSource API      │  - Receives wallet address      │  │
│  │  - Logs     │  (Server-Sent Events) │  - Creates Dome WebSocket       │  │
│  │  - Status   │                       │  - Subscribes to wallet         │  │
│  └─────────────┘                       │  - Streams events to client     │  │
│                                        └────────────┬────────────────────┘  │
│                                                     │                       │
│                                                     │ WebSocket             │
│                                                     ▼                       │
│                                        ┌─────────────────────────────────┐  │
│                                        │         Dome API                │  │
│                                        │    (Polymarket WebSocket)       │  │
│                                        │                                 │  │
│                                        │  - Real-time order events       │  │
│                                        │  - Auto-reconnect               │  │
│                                        │  - Wallet subscription          │  │
│                                        └─────────────────────────────────┘  │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### How It Works

1. **User Input:** Enter any Polymarket wallet address (0x...) in the terminal
2. **SSE Connection:** Browser opens an EventSource connection to `/api/wallet-tracking`
3. **Dome WebSocket:** The API route creates a WebSocket connection to Dome's Polymarket feed
4. **Subscription:** The API subscribes to order events for the specified wallet
5. **Real-time Streaming:** Orders are streamed back to the browser via Server-Sent Events
6. **Live Display:** The terminal displays each order with side (BUY/SELL), price, shares, and market info

### Why SSE + WebSocket?

Browsers can't connect directly to arbitrary WebSocket servers due to CORS and authentication. The architecture uses:

- **Next.js API Route** as a proxy/bridge
- **Dome SDK WebSocket** for real-time Polymarket data
- **Server-Sent Events (SSE)** to stream data to the browser

This pattern keeps the Dome API key secure on the server while providing real-time updates to the client.

## Required Environment Variables

Since this feature runs entirely in the Next.js frontend, you only need to configure the **terminal** environment:

### Terminal Environment (`terminal/.env`)

```env
# Dome API Key - Required for Wallet Tracking
DOME_API_KEY=your_dome_api_key
```

**How to get a Dome API Key:**
1. Go to [https://dashboard.domeapi.io](https://dashboard.domeapi.io)
2. Create an account or sign in
3. Navigate to API Keys section
4. Generate a new API key

> 💡 **Note:** Unlike Market Analysis and Betting Bots, Wallet Tracking does NOT require Supabase Edge Functions. The `DOME_API_KEY` goes directly in your `terminal/.env` file.

## Complete Setup

### 1. Configure Environment

```bash
cd terminal
cp .env.example .env
```

Edit `terminal/.env`:

```env
# Required for Wallet Tracking
DOME_API_KEY=your_dome_api_key

# If you use Super Intelligence or arbitrage from the same terminal, point at Polyback Intelligence:
# INTELLIGENCE_BASE_URL=http://127.0.0.1:8085
```

### 2. Start the Frontend

```bash
cd terminal
bun install
bun run dev
```

### 3. Use the Feature

1. Navigate to [http://localhost:3000/wallet-tracking](http://localhost:3000/wallet-tracking)
2. Enter a Polymarket wallet address (e.g., `0x1234...abcd`)
3. Click **Start Tracking**
4. Watch real-time orders appear in the activity log

## Finding Wallet Addresses

To find interesting wallets to track:

1. **Polymarket Leaderboard:** Check top traders on [polymarket.com/leaderboard](https://polymarket.com/leaderboard)
2. **Polygonscan:** Search for active traders on [polygonscan.com](https://polygonscan.com)
3. **Your Own Wallet:** Track your own trading activity in real-time

## Event Types

The tracker displays the following event types:

| Event | Icon | Description |
|-------|------|-------------|
| **Connected** | ✓ | WebSocket connection established |
| **Subscribed** | ✓ | Successfully subscribed to wallet |
| **Order** | ◆ | Trade executed (BUY or SELL) |
| **Disconnected** | ⚠ | Connection lost (auto-reconnects) |
| **Error** | ✗ | Error occurred |

## Order Information

Each order event includes:

| Field | Description |
|-------|-------------|
| Side | BUY (📈) or SELL (📉) |
| Shares | Number of shares traded |
| Price | Price per share (in cents) |
| Market | Market title or slug |
| Tx Hash | Transaction hash on Polygon |

## Technical Details

### Files Structure

```
terminal/
├── src/
│   ├── app/
│   │   ├── api/
│   │   │   └── wallet-tracking/
│   │   │       └── route.ts          # SSE endpoint with Dome WebSocket
│   │   └── wallet-tracking/
│   │       └── page.tsx              # Page component
│   ├── components/
│   │   └── WalletTrackingTerminal.tsx  # Main terminal UI
│   └── types/
│       └── wallet-tracking.ts        # TypeScript definitions
```

### Dependencies

- `@dome-api/sdk` — Dome API SDK with WebSocket support
- `lucide-react` — Icons (Play, Square, Eye, AlertTriangle)

### Key Implementation Details

**API Route (`/api/wallet-tracking/route.ts`):**
- Creates a ReadableStream for SSE
- Initializes Dome WebSocket with auto-reconnect
- Subscribes to wallet address (lowercase normalized)
- Sends heartbeats every 30 seconds
- Cleans up on client disconnect

**Terminal Component:**
- Uses `EventSource` API for SSE
- Auto-scrolls logs to bottom
- Validates Ethereum address format
- Handles connection lifecycle

## Troubleshooting

| Error | Solution |
|-------|----------|
| "DOME_API_KEY not configured" | Add `DOME_API_KEY` to `terminal/.env` |
| "Invalid wallet address format" | Ensure address is valid Ethereum format (0x + 40 hex chars) |
| "Connection lost" | Click Stop, then Start to reconnect |
| No orders appearing | The wallet may not be actively trading — try a more active wallet |

## Security Notes

- ✅ API key is kept server-side (never exposed to browser)
- ✅ Only reads public order data (no private information)
- ✅ Wallet addresses are public blockchain data

---

← [Back to main README](../../README.md)

