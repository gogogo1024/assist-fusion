## KB × Elasticsearch 本地快速验证

本地用单节点 ES（无鉴权）快速验证 Gateway (本地模式或 RPC 聚合) 的 KB ES 后端。

### 1) 启动 ES（Docker）

```sh
mise run es-up
```

等待 9200 端口就绪（首次需几十秒）。

### 2) 启动 gateway（ES 模式）

```sh
KB_BACKEND=es ES_ADDRS=http://localhost:9200 ES_INDEX=kb_docs HTTP_ADDR=:8081 go run ./services/gateway
```

该任务会设置环境变量：
- KB_BACKEND=es
- ES_ADDRS=http://localhost:9200
- ES_INDEX=kb_docs

### 3) 验证 API

```sh
curl -s -X POST http://localhost:8081/v1/docs \
  -H 'Content-Type: application/json' \
  -d '{"title":"FAQ","content":"重启路由器试试"}'

# 检索
curl -s "http://localhost:8081/v1/search?q=路由器"

# 更新
curl -s -X PUT http://localhost:8081/v1/docs/{id} \
  -H 'Content-Type: application/json' \
  -d '{"content":"请检查网线并重启路由器"}'

# 删除
curl -s -X DELETE http://localhost:8081/v1/docs/{id}
```

### 4) 结束环境

```sh
mise run es-down
```

### 常见问题
- 9200 未就绪：等待容器初始化完成；`docker logs exce-es` 查看启动日志。
- 搜索无结果：确认已写入文档且查询词匹配；或尝试英文/更短词以规避分词差异。
- 中文分词效果一般：默认使用 `standard` analyzer，后续可改用 IK 或 ngram/edge_ngram（见 kb-es-ops.md）。
