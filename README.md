# AssistFusion 客服与知识检索平台（Go / CloudWeGo Hertz）

> 统一单体（Modular Monolith）策略：当前阶段聚焦功能迭代与领域模型清晰度，避免过早拆分微服务。KB 支持内存与 Elasticsearch 双后端，默认尝试 IK 分词，失败回退 n-gram（最小 3）并带 edge_ngram 自动补全。 

## 项目结构（已完成 RPC 重构）

> 历史说明：早期单体入口目录 `services/ticket-svc/` 已在重构后移除（由 `services/gateway/` 统一承担 HTTP 入口 + 聚合角色）。如需查看旧实现，可在 Git 历史中检索该路径。

- `services/gateway/`：统一 HTTP 入口（原单体入口演进）。当设置 `FEATURE_RPC=true` 时作为 BFF 转发到下游 Kitex RPC 服务，否则直接内嵌内存实现。
- `rpc/`
  - `ticket/`
    - `main.go`：Ticket RPC 服务入口（package main）
    - `impl/impl.go`：TicketServiceImpl 业务实现，可被内部工具直接 import（避免 package main 限制）
  - `kb/`：同上（KBServiceImpl）
  - `ai/`：同上（AIServiceImpl）
- `idl/`：Thrift 定义（ticket.thrift / kb.thrift / ai.thrift / common.thrift）
- `kitex_gen/`：统一生成代码目录（多服务共享）
- `internal/`：领域与通用模块（配置 / 仓储 / 观测 / 内存实现等）
- `cmd/rpc-probe/`：内部探针 / 内联启动多个 RPC 进行集成验证（直接 import `rpc/<svc>/impl`）
- `airflow/dags/`：知识入库 DAG 示例
- `docs/chat/`：文档与指南
- `Makefile`：run / build / regen 任务
- `tests/`：单元 / 集成测试（待扩展）

## 快速开始（使用 mise）

前置：已安装 mise（macOS 可用 Homebrew 安装）。

```sh
# 同步依赖
mise run bootstrap

# 启动 gateway（内存 KB，监听 :8081）
mise run run-gateway

# 启动带 Elasticsearch 的 gateway（本地需要 Docker）
mise run es-up
mise run run-gateway-es
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

### Ticket 字段与状态机（扩展）

为适配更细的业务场景，Ticket 增加了若干可选字段，并扩展了状态流转接口（原有接口保持不变，向下兼容）：

新增字段（snake_case）：
- `assignee` 指派人
- `priority` 优先级（字符串，可后续扩展枚举）
- `customer` 客户标识
- `category` 类目
- `tags` 标签数组
- `due_at` 截止时间（秒级时间戳）
- `closed_at` / `canceled_at` 关闭/取消时间

状态机与接口扩展：
- 新增动作端点（PUT）：
  - `/v1/tickets/:id/start` → 进入 `in_progress`（事件 `started`）
  - `/v1/tickets/:id/wait` → 进入 `waiting`（事件 `waiting`）
  - `/v1/tickets/:id/close` → 进入 `closed`（写入 `closed_at`，事件 `closed`）
  - `/v1/tickets/:id/cancel` → 进入 `canceled`（写入 `canceled_at`，事件 `canceled`）
- 既有动作端点保留：`assign` / `escalate` / `resolve` / `reopen`
- 约束（节选）：
  - 已 `resolved/closed/canceled` 的工单禁止 `escalate`（409）
  - `start`/`wait` 不可作用于终态（`resolved/closed/canceled`）（409）
  - `close`/`cancel` 对终态重复操作返回 409
- `assign` 支持请求体包含 `assignee` 与 `note`，会写入 `assignee` 字段并记录事件。

兼容性：老的 JSON 与接口仍可正常工作，新字段均为可选并默认空值；事件与周期（cycles）模型保持不变，仅新增了 `closed_at/canceled_at` 快照字段。

## 前端演示界面（/ui）

内置了一个极简前端用于演示整个业务流程（工单 + 知识库），通过 go:embed 打包在 `services/gateway/public/` 下，随网关进程一起提供静态资源。

运行与访问：

```sh
# 1) 启动 gateway（内存 KB；禁用 Prometheus 9100 端口，避免端口冲突）
PROM_DISABLE=1 KB_BACKEND=memory HTTP_ADDR=:8081 go run ./services/gateway

# 如果 :8081 被占用，可换一个端口
PROM_DISABLE=1 KB_BACKEND=memory HTTP_ADDR=:8083 go run ./services/gateway

# 2) 浏览器访问
# http://localhost:8081/ui  （或对应端口）
```

页面包含：
- 运行诊断：一键请求 `GET /ready`。
- 工单流程：创建工单、指派、升级、解决、重开，查看 `cycles` 与 `events`。
- 知识库：新增/更新/删除文档，搜索与查看 `kb/info`。

### 国际化（i18n）

- 语言切换：页面右上角下拉框（中文 / English），或在 URL 中追加 `?lang=zh-CN|en`。
- 持久化：语言首选项会保存到 `localStorage`，并在后续访问时自动生效。
- 覆盖范围：顶部导航、表格表头、按钮、分页信息、状态徽标、提示/告警、主管看板（统计卡片、未分配/逾期表）等。
- 浏览器语言：默认按浏览器语言自动检测，未命中时回退到中文。

常见问题：
- 端口占用：
  - `:8081` 被占用 → 改用 `HTTP_ADDR=:8083`。
  - Prometheus `:9100` 冲突 → 设置 `PROM_DISABLE=1`（演示/开发环境一般推荐关闭）。
- JSON 大小写：所有接口响应均为 `snake_case`（如 `created_at`、`current_cycle`）。
- RPC 模式：设置 `FEATURE_RPC=true` 可切换为下游 Kitex 模式（需先启动对应 RPC 服务）。

### 坐席工作台使用指南

- 入口：顶部选择“坐席”。默认即进入坐席视图。
- 左侧：筛选队列（状态/关键词）、创建工单、工单列表与分页。
- 右侧：
  - 工单详情与统一操作区：assign/start/wait/escalate/resolve/close/cancel/reopen（支持备注）。
  - 事件时间线/周期查看。
  - 知识库建议：默认以当前工单“标题+描述”为关键词，可修改后搜索，展示前 N 条。
- 提示：右上角会出现轻量提示；失败将显示错误码或原因。

## eino集成说明
- ai-rpc (rpc/ai) 计划集成 eino 编排，支持多 Provider（OpenAI / 火山 / 本地），可通过环境变量 AI_PROVIDER 切换。
- 当前已提供 mock embeddings，无需外部大模型即可本地跑通；Chat 暂返回 not_implemented。

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

## 代码生成（Kitex / Makefile 统一封装）

模块路径：`github.com/gogogo1024/assist-fusion`

推荐使用 Make 统一再生：
```sh
make regen        # 所有服务（ticket / kb / ai）
make regen-ticket # 单个服务
make regen-kb
make regen-ai
```
底层等价 Kitex 命令（仅参考）：
```sh
kitex -module github.com/gogogo1024/assist-fusion -service ticket-rpc idl/ticket.thrift
kitex -module github.com/gogogo1024/assist-fusion -service kb-rpc idl/kb.thrift
kitex -module github.com/gogogo1024/assist-fusion -service ai-rpc idl/ai.thrift
```

实现放入 `impl/` 子目录的原因：
1. 避免与 `main.go` 同目录出现 `package main` 导致无法被其它包导入。
2. 内部工具（`cmd/rpc-probe`、未来的集成测试或基准测试）可直接复用 `New<Service>ServiceImpl()`。
3. 后续若抽象接口层，可在 impl 中扩展多实现（内存 / ES / Mock）并由 main 组装。

### 生成一致性校验

为防止忘记提交生成代码，引入校验：

```sh
make verify-gen   # 若有漂移会退出非 0 并提示先提交 regen 结果
```

CI 建议：在 build 与 test 之后执行 `make verify-gen`，阻止 IDL 变更未 sync 的情况。

## Airflow 使用
DAG 示例草案：`airflow/dags/kb_ingest_dag.py`（文档优先，运行前请按注释调整路径与依赖）。
将文件放入 Airflow 的 dags 目录后，可在 WebUI 中手动触发。

## 检索与分词策略概览

- 内存：可配置 n-gram（默认 2）；查询去重 + 简化 IDF；UTF-8 安全摘要。
- Elasticsearch：优先 IK（ik_max_word + ik_smart 组合字段），失败 => n-gram fallback（min_gram=3, edge_ngram autocomplete）。
- 短查询（<=4 字）增加 autocomplete should 召回，提高匹配质量。

## 延伸阅读

- KB 检索原理（n-gram、倒排索引、IDF 概览）：`docs/chat/guides/kb-search-principles.md`
