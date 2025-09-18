# AssistFusion 客服与知识检索平台（Go / CloudWeGo Hertz）

> 统一单体（Modular Monolith）策略：当前阶段聚焦功能迭代与领域模型清晰度，避免过早拆分微服务。KB 支持内存与 Elasticsearch 双后端，默认尝试 IK 分词，失败回退 n-gram（最小 3）并带 edge_ngram 自动补全。 

## 项目结构

- `services/`
  - `gateway`：统一 HTTP 入口（原 ticket-svc 转型），路由与聚合，FEATURE_RPC=on 时转发至下游 Kitex 服务
  - `ticket-rpc`：工单 RPC 服务（Kitex）
  - `kb-rpc`：知识库 RPC 服务（Kitex）
  - `ai-rpc`：AI / Embeddings RPC 服务（Kitex）
- `internal/common/`：配置、日志、中间件、仓储接口
- `airflow/dags/`：知识入库DAG，调用kb-svc和ai-svc
- `Makefile`：一键启动/测试/依赖管理
- `tests/`：单元测试，默认内存实现
 - `docs/chat/`：文档与指南

## 快速开始（使用 mise）

前置：已安装 mise（macOS 可用 Homebrew 安装）。

```sh
# 同步依赖
mise run bootstrap

# 启动 gateway（内存 KB，监听 :8081）
mise run run-ticket   # 兼容旧命名脚本；后续可重命名为 run-gateway

# 启动带 Elasticsearch 的 gateway（本地需要 Docker）
mise run es-up
mise run run-ticket-es
# 关闭 ES
mise run es-down

# （可选）单独启动各 RPC 服务（示例脚本待补充 run-ticket-rpc / run-kb-rpc / run-ai-rpc）

# 运行测试
mise run test
```

## 运行时探针与核心端点

| Endpoint | 说明 |
|----------|------|
| `GET /health` | 存活探针（进程存活立即 200） |
| `GET /ready` | 就绪探针（若 ES 后端初始化失败可返回 503） |
| `GET /metrics` | Prometheus 指标（Hertz + 进程级） |
| `GET /metrics/domain` | 域内指标细分（业务维度采样） |
| `GET /v1/kb/info` | 知识库后端与 analyzer 模式（memory / es + ik|ngram|standard） |
| `GET /v1/search?q=...&limit=10` | 知识检索，limit 默认 10，上限 50 |

HTTP 响应统一附加：`X-AssistFusion-Project`, `X-AssistFusion-Version` 头部。

## API示例

- Gateway 工单相关
```sh
curl -X POST http://localhost:8081/v1/tickets -d '{"title":"test","desc":"hello"}' -H 'Content-Type: application/json'
```

### 事件与周期（Tickets）

工单采用“按周期建模 + 事件审计流”的方式：

- 周期（Cycles）：每次重开（reopen）都会新增一个处理周期，顶层字段（Status/AssignedAt/ResolvedAt 等）代表“当前周期”的快照；历史周期保存在 `Cycles` 数组中，`CurrentCycle` 指向当前周期索引。
- 事件（Events）：所有关键操作都会写入事件流，便于审计和时间线展示。事件类型包括：`created`、`assigned`、`escalated`、`resolved`、`reopened`。
- 关键约束：已 `resolved` 的工单禁止再 `escalate`（返回 409）。纠错应使用 `reopen`，进入新周期。

常用接口：

```sh
# 1) 获取工单详情（包含 Cycles 与 CurrentCycle 快照）
curl http://localhost:8081/v1/tickets/{id}

# 2) 获取工单所有周期
curl http://localhost:8081/v1/tickets/{id}/cycles

# 3) 获取工单事件流
curl http://localhost:8081/v1/tickets/{id}/events

# 4) 状态流转示例：assign -> escalate -> resolve -> reopen
curl -X PUT http://localhost:8081/v1/tickets/{id}/assign
curl -X PUT http://localhost:8081/v1/tickets/{id}/escalate
curl -X PUT http://localhost:8081/v1/tickets/{id}/resolve
curl -X PUT http://localhost:8081/v1/tickets/{id}/reopen
```

示例响应（摘录，字段已统一 snake_case）：

```json
{
  "id": "...",
  "status": "created",
  "current_cycle": 1,
  "cycles": [
    { "status": "resolved", "created_at": 1694500000, "assigned_at": 1694500100, "resolved_at": 1694500200 },
    { "status": "created",  "created_at": 1694500300 }
  ],
  "events": [
    { "type": "created",  "at": 1694500000 },
    { "type": "assigned", "at": 1694500100 },
    { "type": "escalated","at": 1694500150 },
    { "type": "resolved", "at": 1694500200 },
    { "type": "reopened", "at": 1694500300 }
  ]
}
```
- Gateway 知识库（通过 Gateway 暴露）
```sh
curl -X POST http://localhost:8081/v1/docs -d '{"title":"FAQ","content":"..."}' -H 'Content-Type: application/json'
```
- Gateway AI（Embeddings）
```sh
curl -X POST http://localhost:8081/v1/embeddings -d '{"texts":["客服是什么？"]}' -H 'Content-Type: application/json'
```

## eino集成说明
- ai-svc内置eino编排，支持多Provider（OpenAI/火山/本地），可通过环境变量AI_PROVIDER切换。
- 默认mock provider，无需外部大模型即可本地跑通。

## Airflow使用
- 将`airflow/dags/kb_ingest_dag.py`复制到Airflow的dags目录。
- 按README注释配置data/docs目录和服务地址。
- 仅依赖requests库。

## 环境变量（节选）

| 变量 | 用途 | 示例 |
|------|------|------|
| `HTTP_ADDR` | 服务监听地址 | `:8081` |
| `KB_BACKEND` | 知识库后端选择 | `memory` / `es` |
| `ES_ADDRS` | ES 地址（逗号分隔） | `http://localhost:9200` |
| `ES_INDEX` | ES 索引名 | `kb_docs` |
| `ES_USERNAME` / `ES_PASSWORD` | 安全集群认证 | *(可选)* |
| `AI_PROVIDER` | AI Provider 选择 | `mock` (默认) |
| `AI_API_KEY` | Provider Key | *(可选)* |

未配置 DSN / ES 时自动回退内存实现。

## 依赖（当前精简集）
core: cloudwego/hertz, google/uuid, stretchr/testify
可选 / 规划：elastic/go-elasticsearch, cloudwego/eino, zap

## 文档与指南
- 总览与目录：`docs/chat/README.md`
- 检索与 ES 运维：见 `docs/chat/guides/`

## Elasticsearch 后端快速验证（可选）

前置：本地可用 Docker；默认使用内存 KB，可切换到 ES 后端验证分词与补全。

1) 启动 ES（脚本已内置）
```sh
mise run es-up     # 拉起本地 ES（含必要配置）
```

2) 以 ES 后端启动 Gateway（具体命令以项目脚本为准）
```sh
mise run run-ticket-es
# 或手动设置环境变量：KB_BACKEND=es ES_ADDRS=http://localhost:9200
```

3) 健康检查与后端信息
```sh
curl http://localhost:8081/ready
curl http://localhost:8081/v1/kb/info   # 期望 backend=es，analyzer=ik 或 ngram（fallback）
```

4) 写入与搜索验证（含短查询自动补全）
```sh
curl -X POST http://localhost:8081/v1/docs -H 'Content-Type: application/json' \
  -d '{"title":"安装指南","content":"完整安装操作步骤"}'
curl "http://localhost:8081/v1/search?q=安装&limit=5"   # 短查询应能命中（IK 可用或 ngram 回退 + edge_ngram 自动补全）
```

5) 关闭 ES（需要时）
```sh
mise run es-down
```

## 代码生成（Kitex）

已迁移模块路径：`github.com/gogogo1024/assist-fusion`
示例（在仓库根目录）：
```sh
kitex -module github.com/gogogo1024/assist-fusion -service ticket-rpc idl/ticket.thrift
kitex -module github.com/gogogo1024/assist-fusion -service kb-rpc idl/kb.thrift
kitex -module github.com/gogogo1024/assist-fusion -service ai-rpc idl/ai.thrift
```

## Airflow 使用
DAG 示例草案：`airflow/dags/kb_ingest_dag.py`（文档优先，运行前请按注释调整路径与依赖）。
将文件放入 Airflow 的 dags 目录后，可在 WebUI 中手动触发。

## 检索与分词策略概览

- 内存：可配置 n-gram（默认 2）；查询去重 + 简化 IDF；UTF-8 安全摘要。
- Elasticsearch：优先 IK（ik_max_word + ik_smart 组合字段），失败 => n-gram fallback（min_gram=3, edge_ngram autocomplete）。
- 短查询（<=4 字）增加 autocomplete should 召回，提高匹配质量。

## 延伸阅读

- KB 检索原理（n-gram、倒排索引、IDF 概览）：`docs/chat/guides/kb-search-principles.md`
