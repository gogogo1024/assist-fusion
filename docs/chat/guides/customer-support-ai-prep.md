## 延伸阅读

- KB 检索原理（n-gram、倒排索引、IDF 概览）：[kb-search-principles.md](./kb-search-principles.md)

# 客服系统与AI落地方案准备指南（草案）

日期：2025-09-12
适用范围：本仓库 Go 纯栈方案（CloudWeGo Hertz + Eino + Airflow），使用 mise 作为任务执行器。

## 1. 目标与范围
- 目标：在现有仓库内完成客服系统核心模块（ticket、kb、ai）与数据流程（Airflow）的落地准备，形成可执行的最小方案（MVP）。
- 非目标：不引入外部向量库/消息队列，优先内存/文件方案；不绑定具体云服务商。

## 2. 环境前置
- OS: macOS（zsh）
- 工具链：
  - Go >= 1.22（遵循 `.mise.toml`）
  - Xcode CLT（避免 `dyld: missing LC_UUID`）：`xcode-select --install`
  - mise（用于 run/test/lint/bootstrap）
- 仓库内约定：
  - `internal/common` 提供 config/logger/middleware(含统一错误)/store 基础能力
  - `services/gateway` 提供统一 HTTP 入口（可内联直调或走 RPC）
  - `rpc/<svc>` 提供各领域 Kitex RPC 服务 (ticket / kb / ai)
  - `airflow/` 托管数据编排（DAG）

## 3. 模块与接口
- ticket 域：工单 CRUD、状态流转（assign / escalate / resolve / reopen）、事件与周期记录
- kb 域：文档增删改、关键词检索（内存倒排 + snippet），后续可加 ES / 向量
- ai 域：Embeddings（mock provider 已实现）、Chat TODO（返回 not_implemented）
  - HTTP 通过 gateway 聚合；RPC 通过 kitex_gen/<svc>/*service

## 4. 数据流程（Airflow）
- DAG: `airflow/dags/kb_ingest_dag.py` (草案 / 可选)
  - 读取 `data/docs` 下的文档
  - 清洗/切片（简单分段）
  - 通过 ai-rpc (embeddings) 生成向量（若不可用则跳过）
  - 调用 kb-rpc 写入/更新索引

## 5. MVP 路线图
1) 完成 kb 内存索引 + 搜索（已完成）
2) AI embeddings mock（已完成）
3) ticket 全生命周期事件 & 指标（已完成基础）
4) 添加 Chat 实现与集成测试（进行中 / TODO）
5) 可选：Airflow DAG ingest → embedding → index pipeline

## 6. 运维与开发
- 统一通过 mise / Makefile 运行：
  - 依赖整理：`mise run bootstrap`
  - 启动 RPC：`mise run run-ticket-rpc` / `run-kb-rpc` / `run-ai-rpc`
  - 启动 gateway（直连模式）：`mise run run-gateway`
  - 启动 gateway（RPC 模式）：env FEATURE_RPC=true go run ./services/gateway
  - 生成与校验：`make regen && make verify-gen`
- 日志：zap；请求访问日志已接入
- 故障排查：优先清理 go 缓存、确保 CLT 安装

## 7. 风险与假设
- Eino 版本固定 v0.5.0；若 API 变更需同步升级
- macOS 本地工具链问题可能影响构建/运行
- 内存索引仅适用于小数据量，后续可替换向量库

## 8. 附录
- 目录约定、命名规范、接口草图、DAG 伪代码（稍后补充）
