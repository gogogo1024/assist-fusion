#!/usr/bin/env bash
set -euo pipefail
RUN_NAME="ticket-rpc"
ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
OUT_DIR="$ROOT_DIR/output"
mkdir -p "$OUT_DIR/bin"
cp -r "$ROOT_DIR/script" "$OUT_DIR/" 2>/dev/null || true
chmod +x "$ROOT_DIR"/script/*.sh 2>/dev/null || true
if [[ "${IS_SYSTEM_TEST_ENV:-}" == "1" ]]; then
  go test -c -covermode=set -o "$OUT_DIR/bin/${RUN_NAME}" -coverpkg=./...
else
  go build -o "$OUT_DIR/bin/${RUN_NAME}" .
fi
echo "Built $OUT_DIR/bin/${RUN_NAME}"