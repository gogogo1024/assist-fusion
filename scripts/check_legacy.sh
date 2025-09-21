#!/usr/bin/env bash
set -euo pipefail

legacy_dir="services/ticket-svc"

if [ -d "$legacy_dir" ]; then
  echo "[FAIL] Legacy directory '$legacy_dir' reappeared. This codebase migrated to services/gateway + rpc/* ." >&2
  echo "       Please remove it (git rm -r $legacy_dir) before merging." >&2
  exit 1
fi

echo "[OK] No legacy ticket-svc directory present."