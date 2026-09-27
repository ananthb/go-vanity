#!/usr/bin/env bash
# Builds the Worker into build.
set -euo pipefail
cd "$(dirname "$0")"
go run github.com/syumai/workers-go/cmd/workers-assets-gen -mode=tinygo
tinygo build -o build/app.wasm -target wasm -no-debug -opt=z .
