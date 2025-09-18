## KB 在生产环境使用 Elasticsearch（指南）

本项目的知识库（KB）后端已抽象为 `kb.Repo`，支持内存实现（开发/单测）与 Elasticsearch 实现（生产）。通过环境变量无缝切换。

### 何时选择 ES

- 文档量或 QPS 上升，需要持久化、分片、副本和高可用
- 需要更丰富的检索能力：BM25、短语/布尔/邻近查询、高亮、同义词/停用词、中文/多语言分词等

### 配置项（环境变量）

- `KB_BACKEND`: `memory`（默认）或 `es`
- `ES_ADDRS`: 逗号分隔的地址列表，如 `http://es1:9200,http://es2:9200`
- `ES_INDEX`: 索引名，默认 `kb_docs`
- `ES_USERNAME`, `ES_PASSWORD`: 可选的基本认证

服务启动后，若后端为 `es`，会在首次搜索/写入前自动检查索引，不存在则用最小映射创建：

```json
{
  "mappings": {
    "properties": {
      "title":   {"type": "text", "analyzer": "standard"},
      "content": {"type": "text", "analyzer": "standard"}
    }
  }
}
```

检索侧使用 `multi_match`（`best_fields`），字段权重：`title^2`、`content^1`，并启用内容高亮作为摘要回传。

### 中文与多语言

- 默认 `standard` analyzer 即可基本可用；更高质量建议：
  - 中文：IK Analyzer（需在集群安装插件）或自定义 `ngram/edge_ngram` 分析器
  - 同义词、停用词：在索引分析链中增加相应 filter
  - 以上需要自定义 mapping/template，可将 `ensureIndex` 扩展为显式创建模板/设置

### 运行与运维要点

- 分片/副本：根据数据量与 QPS 规划，建议预估后固定模板
- 刷新与延迟：当前写入使用 `Refresh=true` 便于读一致，生产可按需调整（吞吐 vs 延迟）
- 监控：集群健康、节点/分片状态、查询耗时、拒绝率、慢查询日志
- 安全：鉴权、TLS、基于角色的索引权限
- 生命周期（ILM）：热/温/冷存储策略、快照备份

### 与当前实现的对应关系

- 内存实现中的 n-gram/IDF 与评分被 ES 的分析器与 BM25 替代
- 字段权重（title^2, content^1）在查询层保持一致
- 摘要：使用 ES 高亮作为 snippet；如无高亮则回退全文开头片段

### 故障排查（常见）

- 无法连接：检查 `ES_ADDRS`、网络与鉴权（`ES_USERNAME/PASSWORD`）
- 索引创建失败：索引名冲突或权限不足；查看集群日志
- 中文效果差：确认 analyzer 配置；可先用 `edge_ngram` 过渡

更多背景与原理参见：

- KB 检索原理（n-gram、倒排索引、IDF 概览）：[kb-search-principles.md](./kb-search-principles.md)
