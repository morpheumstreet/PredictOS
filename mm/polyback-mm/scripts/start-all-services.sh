#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

export POLYBACK_CONFIG="${POLYBACK_CONFIG:-$ROOT/configs/develop.yaml}"

echo "=========================================="
echo "Polyback-mm (Go) — all services"
echo "POLYBACK_CONFIG=$POLYBACK_CONFIG"
echo "=========================================="

mkdir -p logs bin
bash -c 'cd "'"$ROOT"'" && make build'

start_bg() {
  local name="$1"
  shift
  echo "Starting $name..."
  nohup "$@" >>"logs/${name}.log" 2>&1 &
  echo $! >"logs/${name}.pid"
  echo "  PID $(cat "logs/${name}.pid")  log logs/${name}.log"
}

echo ""
echo "1. infrastructure-orchestrator (:8084) — Docker; compose under deploy/ (resolved from config path)"
start_bg infrastructure-orchestrator-service "$ROOT/bin/infrastructure" "$POLYBACK_CONFIG"

echo "   Waiting for stacks (adjust sleep if needed)..."
sleep 25

start_bg executor-service "$ROOT/bin/executor" "$POLYBACK_CONFIG"
start_bg strategy-service "$ROOT/bin/strategy" "$POLYBACK_CONFIG"
start_bg ingestor-service "$ROOT/bin/ingestor" "$POLYBACK_CONFIG"
start_bg analytics-service "$ROOT/bin/analytics" "$POLYBACK_CONFIG"
start_bg intelligence-service "$ROOT/bin/intelligence" "$POLYBACK_CONFIG"
start_bg tracker-service "$ROOT/bin/tracker" "$POLYBACK_CONFIG"

echo ""
echo "Health:"
echo "  Executor:       http://localhost:8080/actuator/health"
echo "  Strategy:       http://localhost:8081/actuator/health"
echo "  Analytics:      http://localhost:8082/actuator/health"
echo "  Ingestor:       http://localhost:8083/actuator/health"
echo "  Infrastructure: http://localhost:8084/actuator/health"
echo "  Intelligence:   http://localhost:8085/actuator/health"
echo "  Tracker:        http://localhost:8086/actuator/health"
echo ""
echo "Stop: bash scripts/stop-all-services.sh"
echo ""
