#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
export BROWSERSLIST_IGNORE_OLD_DATA=1
rm -rf dist
mkdir -p dist/assets
bun x --bun tailwindcss -i ./src/globals.css -o ./dist/assets/styles.css --minify
bun build ./src/client/main.tsx --outfile=./dist/assets/main.js --target=browser --minify
cp -R public/. dist/
