#!/usr/bin/env bash
set -euo pipefail

# verify-gen.sh: Detect uncommitted Kitex generated code drift vs IDL.
# Usage: ./script/verify-gen.sh
# Exits non‑zero if running `make regen` would change kitex_gen/ content.

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

if ! command -v kitex >/dev/null 2>&1; then
  echo "[verify-gen] kitex tool not found in PATH" >&2
  exit 2
fi

# Take a snapshot (sha256) of current kitex_gen tracked files.
TMP_BEFORE=$(mktemp)
TMP_AFTER=$(mktemp)

# Only consider files under version control (git ls-files) to avoid noise.
git ls-files 'kitex_gen/**' | sort | while read -r f; do
  if [ -f "$f" ]; then
    shasum -a 256 "$f" >> "$TMP_BEFORE"
  fi
done
sort -o "$TMP_BEFORE" "$TMP_BEFORE"

# Regenerate (uses Makefile targets).
make regen >/dev/null 2>&1 || { echo "[verify-gen] make regen failed" >&2; rm -f "$TMP_BEFORE" "$TMP_AFTER"; exit 3; }

git ls-files 'kitex_gen/**' | sort | while read -r f; do
  if [ -f "$f" ]; then
    shasum -a 256 "$f" >> "$TMP_AFTER"
  fi
done
sort -o "$TMP_AFTER" "$TMP_AFTER"

if ! diff -u "$TMP_BEFORE" "$TMP_AFTER" >/dev/null; then
  echo "[verify-gen] Detected drift in generated kitex_gen code. Run 'make regen' and commit changes." >&2
  echo "--- drift details (before vs after hashes) ---" >&2
  diff -u "$TMP_BEFORE" "$TMP_AFTER" >&2 || true
  rm -f "$TMP_BEFORE" "$TMP_AFTER"
  exit 4
fi

echo "[verify-gen] OK: kitex_gen synchronized with IDL." >&2
rm -f "$TMP_BEFORE" "$TMP_AFTER"

# Guard: ensure manual patched Total field still exists in kb SearchResponse (optional i32 total)
if ! grep -q "type SearchResponse struct" kitex_gen/kb/kb.go; then
  echo "[verify-gen] ERROR: kitex_gen/kb/kb.go missing SearchResponse type (unexpected)" >&2
  exit 5
fi
if ! grep -q "Total *\\*int32" kitex_gen/kb/kb.go; then
  echo "[verify-gen] ERROR: SearchResponse.Total field missing (regeneration may have overwritten manual patch). Re-apply optional total field." >&2
  exit 6
fi

echo "[verify-gen] Field guard: SearchResponse.Total present." >&2
