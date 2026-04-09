# polyback-mm HTTP API

Machine-readable spec: [`openapi.json`](openapi.json) (OpenAPI 3.0.3).

## Hosts and configuration

Ports and bind addresses come from [`configs/develop.yaml`](../configs/develop.yaml) under `server`:

| Service          | Default address | Role                                      |
|------------------|-----------------|-------------------------------------------|
| Executor         | `:8080`         | Polymarket REST, paper/live adapter       |
| Strategy         | `:8081`         | Gabagool engine HTTP + Prometheus         |
| Analytics        | `:8082`         | Status + events stub                      |
| Ingestor         | `:8083`         | Status + WS/Kafka snapshot                |
| Infrastructure   | `:8084`         | Docker compose orchestration              |
| Intelligence     | `:8085`         | Agents, get-events, x402, Polymarket helpers |
| Tracker          | `:8086`         | Polymarket wallet position tracker (Gamma + data-api) |

Override with `POLYBACK_CONFIG` or pass the YAML path as the first argument to each binary.

### Intelligence (`http://127.0.0.1:8085`)

All routes are under **`POST /api/intelligence/<name>`** (same path suffixes as the former Supabase functions): `get-events`, `event-analysis-agent`, `analyze-event-markets`, `bookmaker-agent`, `arbitrage-finder`, `mapper-agent`, `polyfactual-research`, `x402-seller`, `polymarket-put-order`, `polymarket-up-down-15-markets-limit-order-bot`, `polymarket-position-tracker`, plus `POST /api/intelligence/ping`.

**Config file:** [`configs/develop.yaml`](../configs/develop.yaml) keys under `intelligence` (merged with `real.testing.yml` / `real.yml`):

| Key | Purpose | Env if YAML empty | Effective base when still empty |
|-----|---------|-------------------|----------------------------------|
| `intelligence.dome.base_url` | Dome REST base | `DOME_BASE_URL` | `DefaultDomeAPIBaseURL` in `internal/config/api_base_defaults.go` (`https://api.domeapi.io/v1`) |
| `intelligence.dome.api_key` | Dome bearer token | `DOME_API_KEY` | (no default; required for Dome calls) |
| `intelligence.polyfactual.base_url` | Polyfactual API base | `POLYFACTUAL_BASE_URL` | `DefaultPolyfactualAPIBaseURL` |
| `intelligence.polyfactual.api_key` | Polyfactual `X-API-Key` | `POLYFACTUAL_API_KEY` | (no default) |

Whitespace-only `base_url` in YAML is treated as empty and gets the same defaults.

**Related (other YAML sections):** `hft.kalshi_dflow.base_url` → env `DFLOW_BASE_URL` then `DefaultDFlowAPIBaseURL`. `hft.polymarket` gamma / CLOB REST / CLOB WS URLs → defaults in `applyDefaultAPIBaseURLs`. `ingestor.polymarket.data_api_base_url` → `DefaultPolymarketDataAPIBaseURL`. See `internal/config/api_base_defaults.go` for the full list and keep in sync with platform packages.

**Polygon JSON-RPC (unstable public endpoints):** `hft.polymarket.polygon_rpc_urls` lists HTTPS RPC URLs for chain id **`hft.polymarket.chain_id`** (default **137**). When the list is still empty after YAML + env, **`polygon_rpc_chainlist`** can **ingest** HTTPS endpoints from **`https://chainlist.org/rpcs.json`** (see `hft.polymarket.polygon_rpc_chainlist`: `enabled`, `url`, `max_urls`, `timeout_seconds`; env **`POLYGON_RPC_CHAINLIST_URL`**, **`POLYGON_RPC_CHAINLIST_DISABLE`**). If ingest is off or fails, **`DefaultPolygonRPCURLs`** applies. Env for explicit URLs: **`POLYGON_RPC_URLS`** (comma-separated, replaces the list) or **`POLYGON_RPC_URL`** (single URL if the list is still empty). At runtime, **`internal/evmrpc`** probes endpoints in parallel (same idea as [morpheum-labs/pricefeeding/rpcscan](https://github.com/morpheum-labs/pricefeeding/tree/main/rpcscan): `web3_clientVersion`, lowest latency wins) via **`evmrpc.PickFastestRPC`** / **`evmrpc.DialFastest`**. Use **`evmrpc.Manager`** after **`config.Load`** with **`config.NewPolygonEVMRPCManager(root, refresh)`** for a shared client with TTL refresh and **`Invalidate()`** on errors. For production, prefer a paid provider (Alchemy, Infura, etc.). **Full picture:** [polymarket-tracker-and-rpc.md](../../../docs/operations/polyback-mm/polymarket-tracker-and-rpc.md).

Other secrets remain **environment-only** on the intelligence process (e.g. `OPENAI_API_KEY`, `XAI_API_KEY`, `X402_*`, `BLOCKRUN_WALLET_KEY`). The PredictOS terminal proxies to this service via `INTELLIGENCE_BASE_URL`. Wallet lookups for the position tracker use **per-request** `address` / `addresses` on the tracker API; **`POLYMARKET_PROXY_WALLET_ADDRESS`** is an optional default when the body omits wallets (legacy).

### Tracker (`http://127.0.0.1:8086`)

Dedicated process: **`cmd/tracker`**. Design follows the public-data approach of [polymarket-trade-tracker](https://github.com/leolopez007/polymarket-trade-tracker): **Gamma** for market metadata and token ids, **data-api** for `positions` and `activity` (incl. neg-risk style flows), with the same JSON contract as intelligence’s legacy route.

- **`POST /api/tracker/polymarket-position-tracker`** — aligned with **`POST /api/intelligence/polymarket-position-tracker`**. Body: `asset` (BTC, SOL, ETH, XRP), optional `marketSlug`, optional `tokenIds` `{ "up", "down" }`. **Wallets** (same idea as [polymarket-trade-tracker](https://github.com/leolopez007/polymarket-trade-tracker) per-query address): provide **`address`**, **`user`**, or **`wallet`** (single `0x` + 40 hex), or **`addresses`** / **`wallets`** (string arrays). Deduplicated; max 32 per request. If none are sent, **`POLYMARKET_PROXY_WALLET_ADDRESS`** is used when set. **Single wallet** response: `{ data: { asset, walletAddress, position } }`. **Multiple wallets**: `{ data: { asset, wallets: [ { address, success, position? | error? } ] } }`. Uses `hft.polymarket.gamma_url` and `ingestor.polymarket.data_api_base_url` from config (defaults in `internal/config/api_base_defaults.go`).

Point the terminal at this service with a full URL, e.g. `INTELLIGENCE_EDGE_FUNCTION_POSITION_TRACKER=http://127.0.0.1:8086/api/tracker/polymarket-position-tracker`, or keep using intelligence (it delegates to the same `internal/tracker` implementation).

**Implementation layout (SOLID / DRY):** `internal/tracker/port` defines `PolymarketData` and `GammaMarket`; `dataapi` and `gammaadapter` implement them; `market` resolves slug → condition + CLOB legs; `position` holds pure aggregation (no HTTP); root `tracker` package composes dependencies (`NewService` / `NewServiceWith`) and maps HTTP.

**Note:** This stack does **not** expose a WebSocket API to clients. It connects **outbound** to Polymarket CLOB WebSocket. For streaming, consume **Kafka** (`hft.events.topic`, default `polybot.events`) using the envelope in `internal/hftevents/publisher.go` (`ts`, `source`, `type`, `data`). Event types include `market_ws.tob`, `strategy.gabagool.order`, `executor.order.*`.

---

## Common (every service)

### `GET /actuator/health`

Returns `{ "status": "UP" }`.

### `GET /metrics`

Prometheus exposition format (`text/plain`).

---

## Executor (`http://127.0.0.1:8080`)

### `GET /api/polymarket/health`

Query parameters:

- `deep` — `true` to embed cached book when `tokenId` is set.
- `tokenId` — asset id for deep mode.

Response: mode, CLOB URLs, chain id, WS flags, optional `orderBook` (raw JSON) or `deepError`.

### `GET /api/polymarket/account`

Account summary (`mode`, optional address fields).

### `GET /api/polymarket/bankroll`

Equity-style snapshot; paper mode uses configured bankroll.

### `GET /api/polymarket/positions`

Query: `limit`, `offset`.

- **Paper:** JSON array of positions.
- **Live:** `501` (not implemented in Go port yet).

### `GET /api/polymarket/tick-size/{tokenId}`

Returns a JSON number (currently `0.01`).

### `GET /api/polymarket/marketdata/top/{tokenId}`

Cached top of book: `bestBid`, `bestAsk`, sizes, `lastTradePrice`, timestamps.

- **404** if no cache for that token.

### `POST /api/polymarket/orders/limit`

Body (JSON):

| Field               | Type   | Required |
|---------------------|--------|----------|
| `tokenId`           | string | yes      |
| `side`              | `BUY` / `SELL` | yes |
| `price`             | decimal | yes     |
| `size`              | decimal | yes     |
| `orderType`, `tickSize`, `negRisk`, `feeRateBps`, `nonce`, `expirationSeconds`, `taker`, `deferExec` | optional |

- **Paper:** `200` + `OrderSubmissionResult` (`mode`, `clobResponse`, …).
- **Live:** `501` until wired.

### `POST /api/polymarket/orders/market`

Body: `tokenId`, `side`, `amount`, `price`, plus optional fields as above.

### `GET /api/polymarket/orders/{orderId}`

**Paper:** raw order JSON from simulator. **Live:** `501`.

### `DELETE /api/polymarket/orders/{orderId}`

**Paper:** cancel result JSON. **Live:** `501`.

### `GET /api/polymarket/orders` / `GET /api/polymarket/trades`

**501** — not implemented.

---

## Strategy (`http://127.0.0.1:8081`)

### `GET /api/strategy/status`

```json
{ "activeMarkets": 0, "running": true }
```

`running` reflects Gabagool enabled flag in config; `activeMarkets` is live engine count.

---

## Analytics (`http://127.0.0.1:8082`)

### `GET /api/analytics/status`

Returns `app`, `datasourceUrl` (ClickHouse DSN string from config), `eventsTable`.

### `GET /api/analytics/events`

Query: `type` (reserved). Returns `[]` until ClickHouse repository is implemented.

---

## Ingestor (`http://127.0.0.1:8083`)

### `GET /api/ingestor/status`

JSON including: Polymarket username/proxy/API URL, `pollingEnabled`, `marketWsStarted`, `subscribedAssets`, `topOfBookCount`, Kafka topic/enabled flags, Gamma and ClickHouse base URLs.

---

## Infrastructure (`http://127.0.0.1:8084`)

### `GET /api/infrastructure/status`

Compose stack summary: `managed`, `overallHealth`, `stacks[]` with service counts and `healthStatus`.

### `GET /api/infrastructure/health`

Similar payload with `status` (`UP`/`DOWN` style). **503** when `overallHealth` is not `HEALTHY`.

### `GET /api/infrastructure/links`

Human-facing URLs for ClickHouse HTTP/native, Redpanda, Grafana, Prometheus, Alertmanager.

### `POST /api/infrastructure/restart`

Stops then starts all configured stacks. **200** `{ "status": "success", "message": "..." }` or **500** on failure.

---

## Examples

```bash
# Health (executor)
curl -s http://127.0.0.1:8080/actuator/health

# Deep health + TOB cache
curl -s "http://127.0.0.1:8080/api/polymarket/health?deep=true&tokenId=YOUR_TOKEN_ID"

# Strategy
curl -s http://127.0.0.1:8081/api/strategy/status

# Ingestor WS snapshot
curl -s http://127.0.0.1:8083/api/ingestor/status

# Paper limit order
curl -s -X POST http://127.0.0.1:8080/api/polymarket/orders/limit \
  -H 'Content-Type: application/json' \
  -d '{"tokenId":"...","side":"BUY","price":"0.45","size":"10"}'
```

Import **`docs/openapi.json`** into Swagger UI, Postman, or `redoc-cli` for interactive docs.
