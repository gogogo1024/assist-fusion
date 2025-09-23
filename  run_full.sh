#!/usr/bin/env bash
set -e
BASE=${BASE:-http://127.0.0.1:8080}

echo "[KB] Create docs..."
for i in $(seq 1 12); do
  curl -s -X POST $BASE/v1/docs -H 'Content-Type: application/json' \
    -d "{\"title\":\"Doc$i\",\"content\":\"Pagination body $i\"}" > /dev/null
done

echo "[KB] Page1"
curl -s "$BASE/v1/search?q=Doc&limit=5" | jq '{returned,total,next_offset}'
echo "[KB] Page2"
curl -s "$BASE/v1/search?q=Doc&limit=5&offset=5" | jq '{returned,total,next_offset}'
echo "[KB] Page3"
curl -s "$BASE/v1/search?q=Doc&limit=5&offset=10" | jq '{returned,total,next_offset}'
echo "[KB] Info"
curl -s $BASE/v1/kb/info | jq .

echo "[AI] Embeddings"
curl -s -X POST $BASE/v1/embeddings -H 'Content-Type: application/json' \
  -d '{"texts":["hello","world"],"dim":32}' | jq '{dim, vlen:(.vectors[0]|length)}'

echo "[AI] Chat stream (may fallback single JSON if mock)"
curl -N -X POST $BASE/api/ai/chat/stream -H 'Content-Type: application/json' \
  -d '{"messages":[{"role":"user","content":"hi"}]}' | head -n 5

echo "[Ticket] Create"
TID=$(curl -s -X POST $BASE/v1/tickets -H 'Content-Type: application/json' \
  -d '{"title":"TicketFull","desc":"Flow","note":"init"}' | jq -r '.id')
echo "Ticket=$TID"
echo "[Ticket] Assign"
curl -s -X PUT $BASE/v1/tickets/$TID/assign -H 'Content-Type: application/json' -d '{"note":"assign"}' | jq '.status'
echo "[Ticket] Resolve"
curl -s -X PUT $BASE/v1/tickets/$TID/resolve -H 'Content-Type: application/json' -d '{"note":"resolve"}' | jq '.status'
echo "[Ticket] Cycles"
curl -s $BASE/v1/tickets/$TID/cycles | jq '.current'
echo "[Ticket] Events"
curl -s $BASE/v1/tickets/$TID/events | jq '.events|length'

echo "[Observability]"
curl -s $BASE/metrics/domain | head -n 10
echo "[Done]"