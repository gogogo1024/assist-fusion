# 测试策略（草案）

日期：2025-09-12

目标：确保 gateway + (rpc/ticket | rpc/kb | rpc/ai) 的核心路径具备基础回归能力，默认不依赖外部外部系统（ES 可通过内存后端回退）。

## 覆盖范围
- gateway （HTTP 聚合 / 本地模式）
  - 创建/查询/更新/删除（CRUD）
  - 中间件：recover / request-id / access-log（可通过 httptest 断言状态码与头部）
- rpc/kb （KBServiceImpl：文档 CRUD + 搜索）
  - 文档导入（POST /v1/docs）返回 id
  - 关键词搜索（GET /v1/search?q=）返回 items、按 score 排序
- rpc/ai （AIServiceImpl：Embeddings + Chat 占位）
  - embeddings（POST /v1/embeddings）固定维度、可复现（seed）
  - rag（POST /v1/rag）在无外部依赖时可返回基于关键词的占位答案

## 工具与框架
- Go 标准库 testing + net/http/httptest
- stretchr/testify（断言）

## 示例清单（待实现）
- rpc/ticket/impl/impl_test.go
- rpc/kb/impl/impl_test.go
- rpc/ai/impl/impl_test.go
- services/gateway/ticket_http_test.go （集成：HTTP → 本地实现）

## 运行
```sh
go test ./...
# 或仅限某一层：
go test ./rpc/ticket/...
```

## 质量门禁（建议）
- 引入 golangci-lint（.mise 已配置任务）
- 单元测试通过率门槛（例如 > 60%），后续提高

## 延伸阅读

- KB 检索原理（n-gram、倒排索引、IDF 概览）：[kb-search-principles.md](./kb-search-principles.md)
