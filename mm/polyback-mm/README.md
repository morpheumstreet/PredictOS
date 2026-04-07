# polyback-mm

Go implementation of trading services: executor, strategy, ingestor, analytics, and infrastructure orchestrator. Kafka topics and HTTP routes were aligned with the original Java services for compatibility during migration.

## Design (SOLID-oriented)

- **Ports** ([internal/executor/ports](internal/executor/ports)): HTTP depends on `OrderSimulator`, not on `*paper.Simulator` (DIP).
- **Feeds** ([internal/platforms/polymarket/ws/ports.go](internal/platforms/polymarket/ws/ports.go)): `MarketFeed` / `TOBEventEmitter` keep WebSocket code free of full Kafka client concerns (ISP).
- **Wiring** ([internal/wiring](internal/wiring)): composition-only adapters (e.g. `TOBFromPublisher`) live at the edge, not inside domain packages.
- **HTTP** ([internal/executor/httpapi/handler.go](internal/executor/httpapi/handler.go)): `Polymarket` handler + `orderNotifier` separate transport from event publishing (SRP).
- **Metrics**: Prometheus counters are constructed explicitly and injected, not package globals (testability).

## Build

```bash
make tidy   # first time
make build  # writes ./bin/*
go test ./...
```

## Configuration

- Default file: `configs/develop.yaml`
- Override: `export POLYBACK_CONFIG=/path/to/config.yaml` or pass the path as the first argument to any binary.

## HTTP API

- Human-readable: [`docs/API.md`](docs/API.md)
- OpenAPI 3.0: [`docs/openapi.json`](docs/openapi.json)

## Docker Compose (analytics + monitoring)

Compose files, ClickHouse init SQL, and Prometheus/Grafana assets live under [`deploy/`](deploy/). The infrastructure binary resolves them relative to the **polyback-mm repo root**, inferred from your config path (e.g. `configs/develop.yaml` → parent directory).

Optional override (custom layout or CI):

```bash
export POLYBOT_HOME=/absolute/path/to/polyback-mm   # or any directory containing deploy/docker-compose.*.yaml paths as configured
```

You can also set `infrastructure.polybot_home` in YAML to the same effect.

## Run everything (bash)

```bash
bash scripts/start-all-services.sh
bash scripts/stop-all-services.sh
```

Ports match the historical polybot layout: executor `8080`, strategy `8081`, analytics `8082`, ingestor `8083`, infrastructure `8084`.

## Research (Python)

Do **not** port `research/` to Go. Python analysis and tooling live in [../research](../research) (sibling directory under `mm/`). Point it at the same ClickHouse as these services. Optional: symlink `research` into this folder for convenience.

## Status

- **Executor**: paper exchange simulator, core Polymarket REST routes, Kafka event envelope, Prometheus.
- **Strategy**: Gabagool discovery, WS TOB client, quote sizing, order manager, HTTP executor client.
- **Ingestor / analytics**: HTTP status and health; full ingest pipelines and ClickHouse analytics queries are stubs to extend.
- **Crypto**: HMAC and EIP-712 vectors match the original `polybot-core` Java unit tests.

Live CLOB signing and non-paper execution paths return HTTP 501 until wired to a full `PolymarketTradingService` equivalent.
