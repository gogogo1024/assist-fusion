# AssistFusion 微服务演进设计

> 目标：从当前 Hertz 单体演进为 Hertz 网关 + Kitex 后端服务（Ticket / KB / AI），解耦领域、便于独立扩展与弹性。

## 1. 服务拓扑

| 服务 | 技术 | 责任 | 端口建议 |
|------|------|------|----------|
| gateway | Hertz | 对外 HTTP、认证、限流、错误映射、聚合 | :8080 |
| ticket-rpc | Kitex (Thrift) | 工单生命周期、周期/事件审计 | :8201 |
| kb-svc | Kitex (Thrift) | 文档 CRUD、搜索（ES/内存） | :8202 |
| ai-svc | Kitex (Thrift) | Embeddings (及未来 RAG/分类) | :8203 |

## 2. IDL 与错误模型

- 统一 `ServiceError{code,message}`；code 枚举与当前 HTTP 版本保持：`bad_request/not_found/conflict/kb_unavailable/internal_error`。
- 网关映射：
  - `not_found` -> 404
  - `conflict` -> 409
  - `bad_request` -> 400
  - `kb_unavailable` -> 503
  - 其它 / `internal_error` -> 500
- Thrift: 见 `idl/common.thrift` + 各服务 `ticket.thrift` / `kb.thrift` / `ai.thrift`。

## 3. RPC -> HTTP 映射示例

| HTTP Path | RPC 服务/方法 | 说明 |
|-----------|---------------|------|
| POST /v1/tickets | TicketService.CreateTicket | note 可选 |
| GET /v1/tickets/:id | TicketService.GetTicket | 404 -> not_found |
| PUT /v1/tickets/:id/assign | TicketService.Assign | |-|
| PUT /v1/tickets/:id/escalate | TicketService.Escalate | 冲突 409 |
| PUT /v1/tickets/:id/resolve | TicketService.Resolve | 终态 |
| PUT /v1/tickets/:id/reopen | TicketService.Reopen | 新周期 |
| GET /v1/tickets/:id/cycles | TicketService.GetCycles | list<TicketCycle> |
| GET /v1/tickets/:id/events | TicketService.GetEvents | 审计流 |
| POST /v1/docs | KBService.AddDoc | 文档新增 |
| PUT /v1/docs/:id | KBService.UpdateDoc | 局部更新（空字段跳过）|
| DELETE /v1/docs/:id | KBService.DeleteDoc | 204 -> void |
| GET /v1/search?q=..&limit=.. | KBService.Search | limit 上限 50 在网关处理 |
| GET /v1/kb/info | KBService.Info | backend/analyzer |
| POST /v1/embeddings | AIService.Embeddings | 维度可选 |

## 4. 迁移阶段

| 阶段 | 内容 | 验收 |
|------|------|------|
| Phase A | 引入 IDL + 生成 RPC 服务骨架 | 编译通过 / 单测继续绿 |
| Phase B | 网关改造成纯路由层（内部仍调用本地 impl 适配层）| 现有 HTTP 测试不变 |
| Phase C | 网关切换 Kitex 客户端调用 | e2e 测试全部绿 |
| Phase D | 删除旧内嵌实现（ticket/kb/ai 逻辑迁出）| 无遗留引用 |
| Phase E | 增加认证 / 限流 / 链路追踪 | 指标+trace 验收 |

## 5. Observability
- 每个后端服务暴露自身 /metrics（Kitex exporter）。
- 网关聚合并保留 /metrics/domain（保留历史自定义 counters）。
- 追踪（Phase E）：OpenTelemetry（Kitex + Hertz）。

## 6. 配置与注册
- 初期直接配置静态下游地址（env: TICKET_RPC_ADDR, KB_RPC_ADDR, AI_RPC_ADDR）。
- 后续可加入服务发现（Consul / Nacos / etcd）。

## 7. 异常与重试策略
- 网关对幂等读（Get/List/Search/Info/Embeddings）可做一次短重试（超时分类）。
- 非幂等写不自动重试（由客户端或补偿驱动）。

## 8. 数据一致性
- Ticket 目前内存仓储：迁移后抽象 repo -> 可插入 MySQL/Redis。
- KB 搜索：写 -> ES（刷新策略与一致性延迟由客户端预期管理）。

## 9. 后续演进
- RAG Pipeline（ai-svc 增加 RetrieveAndGenerate）
- 事件总线（Kafka）替换当前内存事件流以做异步扩展
- 多租户支持（租户隔离 + 指标按租户打标签）

## 10. 代码生成指引（手动步骤）
```sh
# 安装 kitex (若尚未安装)
go install github.com/cloudwego/kitex/tool/cmd/kitex@latest

# 生成 ticket 服务代码
kitex -module github.com/gogogo1024/assist-fusion -service ticket-rpc idl/ticket.thrift
# 生成 kb 服务
kitex -module github.com/gogogo1024/assist-fusion -service kb-rpc idl/kb.thrift
# 生成 ai 服务
kitex -module github.com/gogogo1024/assist-fusion -service ai-rpc idl/ai.thrift
```

## 11. 回滚策略
- 若 RPC 服务未能稳定上线：gateway 保留原单体模式启动参数（FEATURE_RPC=off）旁路 RPC 调用，直接使用内建内存实现。

---
> 本文档随实现迭代：新增追踪/限流/认证后追加章节。
