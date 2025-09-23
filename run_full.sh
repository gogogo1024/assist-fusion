#!/usr/bin/env bash
set -euo pipefail

# Full end-to-end startup script: Elasticsearch (optional), RPC services, and Gateway.
# Assumptions:
# - You have Go installed
# - (Optional) Docker running if you want ES backend
# - OPENAI_API_KEY optionally exported for real embeddings/chat; else mock provider will be used

# Configurable knobs (override via env before invoking):
: "${HTTP_ADDR:=:8081}"
: "${KB_BACKEND:=memory}"          # memory | es
: "${ES_ADDRS:=http://localhost:9200}"
: "${ES_INDEX:=kb_docs}"
: "${AI_PROVIDER:=mock}"           # mock | openai
: "${OPENAI_EMBED_MODEL:=text-embedding-3-small}"
: "${OPENAI_CHAT_MODEL:=gpt-4o-mini}"
: "${TICKET_RPC_ADDR:=127.0.0.1:8201}"
: "${KB_RPC_ADDR:=127.0.0.1:8202}"
: "${AI_RPC_ADDR:=127.0.0.1:8203}"

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_DIR="$ROOT_DIR/_run_logs"
mkdir -p "$LOG_DIR"

info() { printf "\033[1;32m[INFO]\033[0m %s\n" "$*"; }
warn() { printf "\033[1;33m[WARN]\033[0m %s\n" "$*"; }
err()  { printf "\033[1;31m[ERR ]\033[0m %s\n" "$*"; }

check_port() {
  local p="$1"
  if lsof -iTCP -sTCP:LISTEN -P | grep -q ":${p} "; then
    err "Port :$p already in use. Stop the process or choose another (export HTTP_ADDR / *_RPC_ADDR vars)."
    return 1
  fi
}

# Basic port sanity (does not try to auto-fix)
check_port "${HTTP_ADDR#:}" || exit 1
check_port "${TICKET_RPC_ADDR##*:}" || exit 1
check_port "${KB_RPC_ADDR##*:}" || exit 1
check_port "${AI_RPC_ADDR##*:}" || exit 1

if [[ "$KB_BACKEND" == "es" ]]; then
  if ! curl -s "$ES_ADDRS" >/dev/null 2>&1; then
    warn "Elasticsearch not reachable at $ES_ADDRS. Attempting to start a dev single-node via docker..."
    docker run -d --rm --name af-es \
      -p 9200:9200 -e "discovery.type=single-node" \
      -e "xpack.security.enabled=false" docker.elastic.co/elasticsearch/elasticsearch:8.12.2 >/dev/null
    info "Waiting ES to become ready..."
    for i in {1..30}; do
      if curl -s "$ES_ADDRS" >/dev/null 2>&1; then
        info "Elasticsearch is up."; break
      fi
      sleep 2
      [[ $i -eq 30 ]] && { err "Elasticsearch failed to start"; exit 1; }
    done
  else
    info "Elasticsearch already reachable: $ES_ADDRS"
  fi
fi

# Export env for subprocesses
export HTTP_ADDR KB_BACKEND ES_ADDRS ES_INDEX AI_PROVIDER OPENAI_API_KEY \
  OPENAI_EMBED_MODEL OPENAI_CHAT_MODEL TICKET_RPC_ADDR KB_RPC_ADDR AI_RPC_ADDR

# Start RPC services
info "Starting ticket-rpc at $TICKET_RPC_ADDR";
(
  cd "$ROOT_DIR/rpc/ticket" && nohup go run . >"$LOG_DIR/ticket.log" 2>&1 &
)
info "Starting kb-rpc at $KB_RPC_ADDR";
(
  cd "$ROOT_DIR/rpc/kb" && nohup go run . >"$LOG_DIR/kb.log" 2>&1 &
)
info "Starting ai-rpc at $AI_RPC_ADDR";
(
  cd "$ROOT_DIR/rpc/ai" && nohup go run . >"$LOG_DIR/ai.log" 2>&1 &
)

# Simple wait for RPC health (tcp connect)
wait_tcp() {
  local addr="$1"; local retry=40
  while (( retry-- > 0 )); do
    if nc -z $(echo "$addr" | sed 's#^##'); then return 0; fi
    sleep 0.25
  done
  return 1
}

for a in "$TICKET_RPC_ADDR" "$KB_RPC_ADDR" "$AI_RPC_ADDR"; do
  info "Waiting for $a ..."
  if ! wait_tcp "$a"; then
    err "Service at $a failed to become ready"; exit 1
  fi
  info "$a ready"
done

# Start gateway (Prometheus metrics always enabled on :9100 inside process, plus /metrics)
info "Starting gateway on $HTTP_ADDR"
(
  cd "$ROOT_DIR/services/gateway" && nohup go run . >"$LOG_DIR/gateway.log" 2>&1 &
)

info "All services launched. Logs in $LOG_DIR"
cat <<EOF
Next steps:
  1. Add a KB document:
     curl -X POST http://localhost${HTTP_ADDR}/v1/docs -H 'Content-Type: application/json' -d '{"title":"hello","content":"world"}'
  2. Search:
     curl 'http://localhost${HTTP_ADDR}/v1/search?q=hello'
  3. Vector search (semantic):
     curl 'http://localhost${HTTP_ADDR}/v1/search/vector?q=hello'
  4. Ticket create:
     curl -X POST http://localhost${HTTP_ADDR}/v1/tickets -H 'Content-Type: application/json' -d '{"title":"t1","desc":"d"}'
  5. Check metrics:
     curl http://localhost:9100/metrics | head -n 20
  6. UI:
     open http://localhost${HTTP_ADDR}/ui
EOF
