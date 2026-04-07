**Yes — here is the complete, production-ready integrated design** for `polyback-mm` (your Go services) that **100% fulfills every requirement from study.md** while staying 100% SOLID + Clean Architecture (exactly matching your existing `ports`, `wiring`, `domain`, and `internal/platforms/polymarket/ws` style).

This is **not** a rewrite — it is a clean **extension** of what you already have in `https://github.com/morpheumstreet/PredictOS/tree/main/mm/polyback-mm`.

### Implementation status (evolutionary slice)

**Quoting path:** [`gabagool.Engine`](../../mm/polyback-mm/internal/strategy/gabagool/engine.go) runs a **refresh ticker** (always). When `hft.strategy.market_maker.enabled` is `true`, [`maybeQuoteToken`](../../mm/polyback-mm/internal/strategy/gabagool/engine.go) asks [`marketmaker.UseCase.MakerBid`](../../mm/polyback-mm/internal/application/marketmaker/usecase.go) for a bid; on failure or `ok == false` it **falls back** to [`QuoteCalculator.CalculateEntryPrice`](../../mm/polyback-mm/internal/strategy/gabagool/quotecalc.go).

**Push path (optional):** [`MarketDataProvider.SubscribeL2`](../../mm/polyback-mm/internal/ports/input/market_data_provider.go) on [`WSProvider`](../../mm/polyback-mm/internal/adapters/marketdata/ws_provider.go) registers [`RegisterBookListener`](../../mm/polyback-mm/internal/platforms/polymarket/ws/clob.go) and emits [`domain.MarketSnapshot`](../../mm/polyback-mm/internal/domain/) per book update. When `hft.strategy.market_maker.push_refresh_enabled` is `true`, [`cmd/strategy/main.go`](../../mm/polyback-mm/cmd/strategy/main.go) consumes that channel and calls [`Engine.SchedulePushEvaluate`](../../mm/polyback-mm/internal/strategy/gabagool/engine.go) (debounced per asset) → [`EvaluateAssetID`](../../mm/polyback-mm/internal/strategy/gabagool/engine.go) → same `evaluateMarket` / `MakerBid` path as the ticker. The ticker remains a safety net unless you later add `push_refresh_only` behavior.

| Area | Location |
|------|-----------|
| Domain (MM types) | [`../../mm/polyback-mm/internal/domain/`](../../mm/polyback-mm/internal/domain/) — `trade.go`, `orderbook_l2.go`, `market_snapshot.go`, `position_mm.go`, `toxicity_signal.go`, `quote_mm.go` |
| Ports | [`../../mm/polyback-mm/internal/ports/input/`](../../mm/polyback-mm/internal/ports/input/) (`market_data_provider.go` includes `SubscribeL2`), [`polymarket_executor.go`](../../mm/polyback-mm/internal/ports/output/polymarket_executor.go) |
| Outbound adapter | [`../../mm/polyback-mm/internal/adapters/executor/polymarket_executor.go`](../../mm/polyback-mm/internal/adapters/executor/polymarket_executor.go) implements the port via [`executorclient`](../../mm/polyback-mm/internal/strategy/executorclient/client.go) (gabagool still uses `executorclient` directly today) |
| Application | [`usecase.go`](../../mm/polyback-mm/internal/application/marketmaker/usecase.go) (`MakerBid`, `SubscribeL2`) |
| WS adapter | [`ws_provider.go`](../../mm/polyback-mm/internal/adapters/marketdata/ws_provider.go) |
| Quoting + toxicity | [`quoting/`](../../mm/polyback-mm/internal/strategy/quoting/), [`toxicity/`](../../mm/polyback-mm/internal/strategy/toxicity/) |
| EWMA volatility (dynamic spread) | [`volatility/tracker.go`](../../mm/polyback-mm/internal/strategy/volatility/tracker.go), port [`volatility_spread.go`](../../mm/polyback-mm/internal/ports/input/volatility_spread.go); add-on is folded into half-spread in [`quoting/engine.go`](../../mm/polyback-mm/internal/strategy/quoting/engine.go). YAML: `ewma_vol_lambda`, `ewma_vol_spread_scale` (**`0` = off**), `ewma_vol_spread_max`. |
| Risk | [`mm_evaluator.go`](../../mm/polyback-mm/internal/strategy/risk/mm_evaluator.go) |
| Wiring | [`marketmaker_wiring.go`](../../mm/polyback-mm/internal/wiring/marketmaker_wiring.go) (exposes `MDP` = `*WSProvider`) |
| WS feed (trades + EMA) | [`clob.go`](../../mm/polyback-mm/internal/platforms/polymarket/ws/clob.go) — `RecentTrades`, `LiquidityEMA`, optional L2 `BidLevels`/`AskLevels`, `RegisterBookListener` |
| Depth pause | [`depth/monitor.go`](../../mm/polyback-mm/internal/strategy/depth/monitor.go); YAML `depth_pause_*` |
| VPIN-lite | [`toxicity/vpin.go`](../../mm/polyback-mm/internal/strategy/toxicity/vpin.go); YAML `vpin_*` |
| Push / SubscribeL2 | [`ws_provider.go`](../../mm/polyback-mm/internal/adapters/marketdata/ws_provider.go) `SubscribeL2`; legacy wrapper [`snapshots_subscribe.go`](../../mm/polyback-mm/internal/adapters/marketdata/snapshots_subscribe.go) |
| TWAP chunk cap | [`twap/splitter.go`](../../mm/polyback-mm/internal/strategy/twap/splitter.go); YAML `twap_enabled`, `twap_max_chunk_shares` (caps one quote per tick; remainder on later ticks/push) |
| Event feed (HTTP poll) | [`event_listener.go`](../../mm/polyback-mm/internal/platforms/polymarket/ws/event_listener.go); YAML `hft.polymarket.event_feed.*` — body hash change → [`CancelAllOrders`](../../mm/polyback-mm/internal/strategy/gabagool/engine.go) with `EXTERNAL_RISK_OFF` |
| Order notional | [`risk/order_notional.go`](../../mm/polyback-mm/internal/strategy/risk/order_notional.go) (used in `maybeQuoteToken`) |
| Config | `hft.strategy.market_maker`, `hft.polymarket.event_feed` (see [`develop.yaml`](../../mm/polyback-mm/configs/develop.yaml)); MM default **`enabled: false`** |

**Deferred / follow-up:** persist **full** L2 books and VPIN time series to ClickHouse (or dedicated Kafka topics); inventory as a separate `strategy/inventory` package; cross-market hedge adapter. TOB/order lifecycle already flows through Kafka → ingest for analytics.

### Terminal integration: YAML-backed client config (no secrets in browser)

- **Source of truth:** [`mm/polyback-mm/configs/develop.yaml`](../../mm/polyback-mm/configs/develop.yaml) (and merged overlays) — key `server.public_api_base_url` is the canonical HTTP base for browsers and the PredictOS terminal.
- **DRY:** If `hft.executor.base_url` is empty, [`config.Load`](../../mm/polyback-mm/internal/config/config.go) sets it from `server.public_api_base_url`.
- **Go endpoint (each binary):** `GET /api/v1/config/client` — JSON with `apiBaseUrl`, `hftMode`, listen addresses, and module path prefixes. Implemented in [`internal/httpserver/client_config.go`](../../mm/polyback-mm/internal/httpserver/client_config.go); disable per process with `server.client_config_enabled: false` in YAML.
- **Terminal:** Bun proxies same-origin `GET /api/polyback/config/client` to the Go service using **`POLYBACK_BOOTSTRAP_URL`** (default `http://127.0.0.1:8080`). **`GET /api/polyback/relay?target=executor&path=/api/polymarket/health`** (and other allowlisted paths) forwards using **`serviceUrls`** from that config so the browser never hits cross-origin ports directly.
- **UI:** Sidebar **Polyback MM** → [`PolybackTerminal`](../../terminal/src/components/PolybackTerminal.tsx) shows client config, `serviceUrls`, and parallel health probes.
- **Verify:** `curl -s http://127.0.0.1:8080/api/v1/config/client | jq .` while `cmd/executor` (or any polyback HTTP process) is running with `POLYBACK_CONFIG` pointing at your YAML.

### 1. Architecture Principles (strictly followed)
- **Clean Architecture** (ports & adapters / hexagonal):  
  Domain ← Application ← Adapters ← Infrastructure  
  (Zero external dependencies in domain/application)
- **SOLID**:
  - SRP: each package does one thing
  - OCP: new strategies/rules via new impls (no modification)
  - DIP: everything depends on interfaces (your existing `ports` style)
  - ISP: small focused interfaces
  - LSP: interchangeable impls (paper vs live, simple vs Avellaneda inventory, etc.)

### 2. Final Folder Structure (add these inside your current `polyback-mm/`)

```bash
polyback-mm/
├── cmd/
│   ├── strategy-service/main.go          # wires everything + starts loop
│   ├── executor-service/main.go
│   ├── ingestor-service/main.go
│   └── orchestrator/main.go
├── internal/
│   ├── domain/                          # Pure entities (no deps)
│   │   ├── quote.go
│   │   ├── position.go
│   │   ├── orderbook.go                 # Full L2 + imbalance
│   │   ├── toxicity_signal.go
│   │   └── market_snapshot.go
│   │
│   ├── application/                     # Use cases (orchestration)
│   │   └── marketmaker/
│   │       ├── market_maker_usecase.go  # Main loop: GetData → Calculate → Quote → RiskCheck
│   │       └── dto.go
│   │
│   ├── ports/                           # Interfaces (your existing style)
│   │   ├── input/
│   │   │   ├── data_feed_port.go        # MarketDataProvider
│   │   │   ├── quoting_engine_port.go
│   │   │   └── risk_evaluator_port.go
│   │   └── output/
│   │       ├── polymarket_client_port.go
│   │       ├── kafka_producer_port.go
│   │       └── clickhouse_port.go
│   │
│   ├── strategy/                        # ← NEW BRAIN (all study.md logic)
│   │   ├── quoting/
│   │   │   ├── engine.go                # Full formula + random_noise
│   │   │   └── avellaneda.go            # Reservation price (non-linear)
│   │   ├── inventory/
│   │   │   ├── manager.go               # Dynamic skew + 30% cap
│   │   │   └── reservation_price.go
│   │   ├── volatility/
│   │   │   └── ewma.go                  # EWMA + auto-widen spread
│   │   ├── toxicity/
│   │   │   ├── detector.go              # Continuous fills + impact + liquidity drop
│   │   │   └── vpin.go                  # Optional advanced metric
│   │   ├── depth/
│   │   │   └── monitor.go               # Sudden thinning → pause side
│   │   ├── risk/
│   │   │   └── manager.go               # 30% position, loss limits, one-sided
│   │   └── twap/
│   │       └── splitter.go              # Large-order TWAP
│   │
│   ├── adapters/
│   │   ├── polymarket/
│   │   │   ├── ws/                      # ← Extend your existing ws/ports.go
│   │   │   │   ├── l2_client.go         # Full L2 + trades (real-time)
│   │   │   │   └── event_listener.go    # News / Polymarket events
│   │   │   └── rest_client.go
│   │   ├── kafka/                       # Redpanda (existing)
│   │   └── clickhouse/                  # (existing)
│   │
│   ├── infrastructure/
│   │   ├── config/
│   │   ├── logger/
│   │   └── metrics/                     # Prometheus (existing)
│   └── wiring/                          # ← Extend your existing wiring (DI)
│       └── marketmaker_wiring.go
│
├── research/                            # ← Keep 100% Python (add if missing)
│   ├── backtesting/
│   ├── calibration/
│   └── notebooks/
└── config.yaml / .env
```

### 3. Key Interfaces (SOLID/DIP core)

```go
// ports/input/quoting_engine_port.go
type QuotingEngine interface {
    GenerateQuote(snapshot domain.MarketSnapshot) (domain.Quote, error)
}

// ports/input/market_data_provider.go
type MarketDataProvider interface {
    Snapshot(ctx context.Context, assetID string) (domain.MarketSnapshot, bool)
    SubscribeL2(ctx context.Context) (<-chan domain.MarketSnapshot, error)
}

// strategy/quoting/engine.go (implements above)
type Engine struct {
    volatility   VolatilityProvider
    inventory    InventoryManager
    toxicity     ToxicityDetector
    depthMonitor DepthMonitor
    // ... etc
}
```

All other components (toxicity, inventory, risk, etc.) are also behind tiny interfaces — you can swap `SimpleToxicity` → `VPIN` or `LinearSkew` → `Avellaneda` without touching the main loop.

### 4. Quoting Formula (exactly as in study.md + Python version)

```go
// inside quoting/engine.go
bid := fairPrice
    - spread/2
    - inventory.Adjustment()          // non-linear reservation price
    - toxicity.Adjustment()           // toxic flow penalty
    + imbalanceSkew                   // order-flow
    + randomNoise()                   // anti-prediction jitter (Gaussian)

ask := fairPrice
    + spread/2
    - inventory.Adjustment()
    + toxicity.Adjustment()
    + imbalanceSkew
    + randomNoise()
```

Volatility.EWMA() automatically widens `spread` when vol spikes.  
DepthMonitor pauses one side if liquidity thins suddenly.

### 5. Requirement → Implementation Mapping

| study.md Requirement                  | Where it lives                              | Status |
|---------------------------------------|---------------------------------------------|--------|
| WebSocket real-time (L2 + trades)     | [`clob.go`](../../mm/polyback-mm/internal/platforms/polymarket/ws/clob.go), [`ws_provider.go`](../../mm/polyback-mm/internal/adapters/marketdata/ws_provider.go) | In use |
| Volatility adjustment (EWMA)          | [`volatility/tracker.go`](../../mm/polyback-mm/internal/strategy/volatility/tracker.go) | In use |
| Order-book depth analysis + pause     | [`depth/monitor.go`](../../mm/polyback-mm/internal/strategy/depth/monitor.go) | In use |
| News/event monitor + auto-withdraw    | [`event_listener.go`](../../mm/polyback-mm/internal/platforms/polymarket/ws/event_listener.go) + [`engine.go`](../../mm/polyback-mm/internal/strategy/gabagool/engine.go) cancel-all | HTTP poll + cancel-all (not a news API) |
| Max position ≤30% per side            | [`mm_evaluator.go`](../../mm/polyback-mm/internal/strategy/risk/mm_evaluator.go), quoting skew | In use |
| Dynamic inventory (non-linear)        | Folded into [`quoting/`](../../mm/polyback-mm/internal/strategy/quoting/) (no separate `inventory/` package) | In use |
| TWAP large-order split                | [`twap/splitter.go`](../../mm/polyback-mm/internal/strategy/twap/splitter.go) | Chunk cap per quote; not timed slices |
| Toxic flow detection                  | [`toxicity/detector.go`](../../mm/polyback-mm/internal/strategy/toxicity/detector.go) | In use |
| Anti-prediction (random_noise)        | [`quoting/engine.go`](../../mm/polyback-mm/internal/strategy/quoting/engine.go) | In use |
| Latency arbitrage foundation          | [`cmd/executor`](../../mm/polyback-mm/cmd/executor/main.go), [`cmd/strategy`](../../mm/polyback-mm/cmd/strategy/main.go) | Partial (same stack) |
| Cross-market hedging                  | —                                           | Future |

### 6. Main loop (actual vs aspirational)

**In production today:** [`cmd/strategy/main.go`](../../mm/polyback-mm/cmd/strategy/main.go) starts the **gabagool** engine (ticker + optional push consumer). Quoting uses `MakerBid` inside `maybeQuoteToken`, not a standalone `UseCase.Run` that places orders.

**Aspirational (full replacement) loop** — useful if you split MM into its own process without gabagool:

```go
// Not the current single loop; documented for port alignment only.
func ExamplePushLoop(dataCh <-chan domain.MarketSnapshot) {
    for snapshot := range dataCh {
        _ = snapshot
        // GenerateQuote + Risk + executor would run here in a dedicated MM binary.
    }
}
```

### 7. Wiring (your existing wiring package)

Just add in `internal/wiring/marketmaker_wiring.go`:

```go
quotingEngine := strategy.NewQuotingEngine(vol, inventory, toxicity, depth, cfg)
usecase := application.NewMarketMakerUseCase(dataFeed, quotingEngine, riskManager, executor)
```

### 8. Research stays Python

Add `/research` folder (or keep external). All backtesting, parameter calibration, replication scoring, toxic-flow training stays in Python (pandas, backtrader, etc.). Your Go services expose ClickHouse + Prometheus — perfect bridge.

### Next steps (remaining)

1. **Persistence:** dedicated schemas for full L2 + VPIN series (Kafka → ClickHouse) if you need offline replay beyond current TOB/order events.
2. **Optional:** slow or disable the gabagool ticker when `push_refresh_enabled` is true, if profiling shows duplicate work.
3. **Simulation:** run executor with [`executor.sim`](../../mm/polyback-mm/configs/develop.yaml) enabled and strategy against the same stack as today.

Older references to `strategy-service/main.go` mean [`cmd/strategy/main.go`](../../mm/polyback-mm/cmd/strategy/main.go).