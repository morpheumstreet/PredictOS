#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
export BROWSERSLIST_IGNORE_OLD_DATA=1
mkdir -p public/assets
bun x --bun tailwindcss -i ./src/globals.css -o ./public/assets/styles.css
bun build ./src/client/main.tsx --outdir=./public/assets --target=browser --sourcemap=external
bun x --bun tailwindcss -i ./src/globals.css -o ./public/assets/styles.css --watch &
TW_PID=$!
bun build ./src/client/main.tsx --outdir=./public/assets --target=browser --sourcemap=external --watch &
BD_PID=$!
trap 'kill "$TW_PID" "$BD_PID" 2>/dev/null; exit 0' EXIT INT TERM
# Verifiable Irys uploads: set IRYS_UPLOAD_SERVICE_URL and run the Go service in another shell:
#   cd mm/irys-upload && go run ./cmd/irys-upload
exec bun --hot ./server.ts
