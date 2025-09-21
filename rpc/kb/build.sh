#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "$0")" && pwd)
SERVICE=kb-rpc
OUT_DIR=$ROOT_DIR/output
BIN_DIR=$OUT_DIR/bin
mkdir -p "$BIN_DIR"

echo "Building $SERVICE ..."
if [[ "${IS_SYSTEM_TEST_ENV:-0}" == "1" ]]; then
  go test -c -o "$BIN_DIR/$SERVICE" ./...
else
  go build -o "$BIN_DIR/$SERVICE" ./...
fi

cp -r "$ROOT_DIR/script" "$OUT_DIR" 2>/dev/null || true

echo "Build artifact at $BIN_DIR/$SERVICE"
