#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

for svc in infrastructure-orchestrator-service executor-service strategy-service ingestor-service analytics-service intelligence-service; do
  pidfile="logs/${svc}.pid"
  if [[ -f "$pidfile" ]]; then
    pid="$(cat "$pidfile")"
    if kill -0 "$pid" 2>/dev/null; then
      echo "Stopping $svc (PID $pid)"
      kill "$pid" || true
    fi
    rm -f "$pidfile"
  fi
done

echo "Done."
