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
  - `services/*` 提供各微服务实现
  - `airflow/` 托管数据编排（DAG）

## 3. 模块与接口
- ticket-svc（已有）：工单 CRUD、SLA、中间件（recover、reqID、accessLog）
- kb-svc（计划）：
  - POST /kb/docs 导入文档（支持 txt/markdown）
  - GET  /kb/search?q= 关键词检索（先内存倒排索引）；后续对接 ai-svc embeddings
- ai-svc（计划，基于 Eino v0.5.0）：
  - POST /embeddings 生成嵌入（默认 mock provider，可替换）
  - POST /rag 基于 kb 做简单检索增强回答（RAG）
  - POST /classify 工单意图分类（可选）

## 4. 数据流程（Airflow）
- DAG: `airflow/dags/kb_ingest_dag.py`
  - 读取 `data/docs` 下的文档
  - 清洗/切片（简单分段）
  - 通过 ai-svc 生成 embeddings（若不可用则跳过）
  - 调用 kb-svc 写入/更新索引

## 5. MVP 路线图
1) 补全 kb-svc（内存索引 + 关键词搜索）
2) ai-svc 提供 mock embeddings 接口（固定维度/可重复）
3) Airflow DAG 打通 ingest → embedding → index pipeline
4) 为 ticket-svc 与 kb-svc 增加 httptest

## 6. 运维与开发
- 统一通过 mise 运行：
  - 依赖整理：`mise run bootstrap`
  - 运行 ticket：`mise run run-ticket`
  - 后续新增：`run-kb`、`run-ai`、`run-airflow`
- 日志：zap；请求访问日志已接入
- 故障排查：优先清理 go 缓存、确保 CLT 安装

## 7. 风险与假设
- Eino 版本固定 v0.5.0；若 API 变更需同步升级
- macOS 本地工具链问题可能影响构建/运行
- 内存索引仅适用于小数据量，后续可替换向量库

## 8. 附录
- 目录约定、命名规范、接口草图、DAG 伪代码（稍后补充）
