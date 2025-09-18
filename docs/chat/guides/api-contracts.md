## 延伸阅读

- KB 检索原理（n-gram、倒排索引、IDF 概览）：[kb-search-principles.md](./kb-search-principles.md)

# 接口契约（草案）

日期：2025-09-12

本文描述 kb-svc 与 ai-svc 的最小可用接口，便于后续实现与对接。

## ticket-svc
- Base: http://localhost:8081
- 数据模型（摘要）
  - Ticket
    - 顶层快照：ID, Title, Desc, Status, CreatedAt, AssignedAt, ResolvedAt, EscalatedAt, ReopenedAt
    - 周期与审计：Cycles: TicketCycle[], CurrentCycle: number, Events: TicketEvent[]
  - TicketCycle: { CreatedAt, AssignedAt, ResolvedAt, EscalatedAt, Status }
  - TicketEvent: { Type, At, Note? }
- 状态机（约束）
  - created → assigned → (optional) escalated → resolved
  - resolved 后禁止 escalate（返回 409）
  - reopen 仅允许从 resolved 进入，reopen 会新增一个新的处理周期（CurrentCycle 指向新周期），顶层快照回到 created
- Endpoints
  - POST /v1/tickets
    - Request: { title: string, desc: string }
    - Response: Ticket（包含快照、Cycles、CurrentCycle、Events）
  - GET /v1/tickets
    - Response: Ticket[]（简化列表）
  - GET /v1/tickets/:id
    - Response: Ticket（包含 Cycles 与 CurrentCycle）
  - PUT /v1/tickets/:id/assign → 200
  - PUT /v1/tickets/:id/escalate → 200；若已 resolved → 409
  - PUT /v1/tickets/:id/resolve → 200（会清空顶层 EscalatedAt 并将当前周期 EscalatedAt 清零）
  - PUT /v1/tickets/:id/reopen → 200；若非 resolved → 409（新增周期，顶层快照回到 created）
  - GET /v1/tickets/:id/cycles → 200
    - Response: { current: number, cycles: TicketCycle[] }
  - GET /v1/tickets/:id/events → 200
    - Response: { events: TicketEvent[] }（按时间顺序：created, assigned, escalated, resolved, reopened, ...）
  - 备注（note）：上述创建与变更接口均可在请求体传入可选字段 { note?: string }，会记录到对应事件的 Note 字段。

示例（可复制运行）：

```sh
# Base
BASE=http://localhost:8081

# 创建工单
curl -s -X POST "$BASE/v1/tickets" -H 'Content-Type: application/json' \
  -d '{"title":"demo","desc":"hello","note":"from README"}' | tee /tmp/ticket.json

# 提取工单ID（jq 可选；若无 jq，可手动查看 /tmp/ticket.json）
ID=$(jq -r .ID </tmp/ticket.json 2>/dev/null || sed -n 's/.*"ID":"\([^"]*\)".*/\1/p' /tmp/ticket.json)
echo ID=$ID

# 状态流转：assign -> escalate -> resolve -> reopen（带 note）
curl -s -X PUT "$BASE/v1/tickets/$ID/assign"   -H 'Content-Type: application/json' -d '{"note":"assigning"}' | cat > /dev/null
curl -s -X PUT "$BASE/v1/tickets/$ID/escalate" -H 'Content-Type: application/json' -d '{"note":"urgent"}'   | cat > /dev/null
curl -s -X PUT "$BASE/v1/tickets/$ID/resolve"  -H 'Content-Type: application/json' -d '{"note":"fixed"}'    | cat > /dev/null
curl -s -X PUT "$BASE/v1/tickets/$ID/reopen"   -H 'Content-Type: application/json' -d '{"note":"wrong fix"}' | tee /tmp/ticket-reopened.json > /dev/null

# 周期与事件
curl -s "$BASE/v1/tickets/$ID/cycles" | tee /tmp/ticket-cycles.json
curl -s "$BASE/v1/tickets/$ID/events" | tee /tmp/ticket-events.json
```

## kb-svc
- Base: 当前由 ticket-svc 暴露 (http://localhost:8081)
- 语义说明
  - 文档写入为 Upsert：同一 ID 再次写入会原子性替换旧内容，同时撤销旧内容在倒排索引中的贡献，避免索引泄漏。
  - 检索：
    - 主路径基于 n-gram（默认 bigram）倒排索引，标题权重高于正文；对查询 n-gram 去重并使用简化 IDF 加权（常见 gram 权重更低）。
    - 无索引命中时回退到子串匹配（标题 +2，正文 +1）。
  - 摘要 snippet 为 UTF-8 安全截断（按 rune 截断，默认最多 120 个字符）。
- Endpoints
  - POST /v1/docs
    - Request: { title: string, content: string, meta?: object }
    - Response: { id: string }
  - GET /v1/search?q=keyword&limit=10
    - Response: { items: Array<{ id: string, title: string, snippet: string, score: number }>, total: number }

示例：

```sh
BASE=http://localhost:8081
curl -s -X POST "$BASE/v1/docs" -H 'Content-Type: application/json' \
  -d '{"title":"FAQ 客服","content":"客服如何升级？请参考SLA"}'
curl -s "$BASE/v1/search?q=客服&limit=10" | tee /tmp/kb-search.json
```

## ai-svc
- Base: http://localhost:8083
- Endpoints
  - POST /v1/embeddings
    - Request: { texts: string[], model?: string }
    - Response: { vectors: number[][], dim: number, model: string }
  - POST /v1/rag
    - Request: { query: string, top_k?: number }
    - Response: { answer: string, references: Array<{ id: string, title: string, score: number }> }
  - POST /v1/classify
    - Request: { text: string, labels: string[] }
    - Response: { label: string, scores: Record<string, number> }

示例：

```sh
BASE=http://localhost:8081
curl -s -X POST "$BASE/v1/embeddings" -H 'Content-Type: application/json' \
  -d '{"texts":["hello"],"dim":4}' | tee /tmp/embeddings.json
```
## 错误约定
- 统一错误格式：{ code: string, message: string, request_id?: string }
- HTTP 状态码：4xx 客户端错误；5xx 服务端错误

## 安全与速率限制（后续）
- 通过中间件添加 IP 限频与简单鉴权（API Key）
- 生产环境使用 HTTPS 及更细粒度鉴权
