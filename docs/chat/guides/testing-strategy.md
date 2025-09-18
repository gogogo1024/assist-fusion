# 测试策略（草案）

日期：2025-09-12

目标：确保 ticket-svc、kb-svc、ai-svc 的核心路径具备基础回归能力，默认不依赖外部服务（使用内存实现/Mock）。

## 覆盖范围
- ticket-svc
  - 创建/查询/更新/删除（CRUD）
  - 中间件：recover / request-id / access-log（可通过 httptest 断言状态码与头部）
- kb-svc（计划）
  - 文档导入（POST /v1/docs）返回 id
  - 关键词搜索（GET /v1/search?q=）返回 items、按 score 排序
- ai-svc（计划）
  - embeddings（POST /v1/embeddings）固定维度、可复现（seed）
  - rag（POST /v1/rag）在无外部依赖时可返回基于关键词的占位答案

## 工具与框架
- Go 标准库 testing + net/http/httptest
- stretchr/testify（断言）

## 示例清单（待实现）
- services/ticket-svc/handlers/tickets_test.go
- services/kb-svc/handlers/docs_test.go
- services/ai-svc/handlers/embeddings_test.go

## 运行
```sh
mise run test
```

## 质量门禁（建议）
- 引入 golangci-lint（.mise 已配置任务）
- 单元测试通过率门槛（例如 > 60%），后续提高

## 延伸阅读

- KB 检索原理（n-gram、倒排索引、IDF 概览）：[kb-search-principles.md](./kb-search-principles.md)
