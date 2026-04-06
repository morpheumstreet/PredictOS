# polyback-mm

Go port of [polybot-main](../polybot-main) trading services: executor, strategy, ingestor, analytics, and infrastructure orchestrator. Kafka topic and HTTP routes follow the Java services so you can mix Go and Java during migration.

## Build

```bash
make tidy   # first time
make build  # writes ./bin/*
go test ./...
```

## Configuration

- Default file: `configs/develop.yaml`
- Override: `export POLYBACK_CONFIG=/path/to/config.yaml` or pass the path as the first argument to any binary.

## Polybot checkout (Docker Compose)

The infrastructure service runs `docker compose` using YAML files from the Java repo. Set:

```bash
export POLYBOT_HOME=/path/to/mm/polybot-main
```

If unset, the code tries `../polybot-main` relative to the **current working directory** (works when you run from this directory next to `polybot-main`).

## Run everything (bash)

```bash
bash scripts/start-all-services.sh
bash scripts/stop-all-services.sh
```

Ports match the Java layout: executor `8080`, strategy `8081`, analytics `8082`, ingestor `8083`, infrastructure `8084`.

## Research (Python)

Do **not** port `research/` to Go. Keep using the Python tree under `polybot-main/research/` (same ClickHouse as these services). Optional: add a symlink `research -> ../polybot-main/research` in this folder if you want a single working tree.

## Status

- **Executor**: paper exchange simulator, core Polymarket REST routes, Kafka event envelope, Prometheus.
- **Strategy**: Gabagool discovery, WS TOB client, quote sizing, order manager, HTTP executor client.
- **Ingestor / analytics**: HTTP status and health; full Java ingest pipelines and ClickHouse analytics queries are stubs to extend.
- **Crypto**: HMAC and EIP-712 vectors match `polybot-core` unit tests.

Live CLOB signing and non-paper execution paths return HTTP 501 until wired to a full `PolymarketTradingService` equivalent.
