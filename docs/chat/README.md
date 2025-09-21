## 文档总览（chat 子目录）

该目录聚合：
1. 架构可视化（C4 图）
2. 设计 / 运行 / 测试 指南（guides/*）
3. 历史会话与决策复盘（session*.md，可选保留）

### 结构

```
docs/chat/
	README.md                # 当前文件（索引）
	architecture/            # C4 Container 等图形（.md/.puml/.svg）
	guides/                  # 专题指南（API 契约 / ES 运维 / 测试 / Runbook 等）
	session-*.md             # 可选：会话 / 迁移复盘
```

### 与代码生成相关的开发流程

修改 / 新增 Thrift IDL → 本地再生产 → 校验无漂移 → 提交：

```sh
# 1. 编辑 idl/*.thrift
vim idl/ticket.thrift

# 2. 重新生成
make regen

# 3. 校验生成代码是否同步（无输出差异即通过）
make verify-gen

# 4. 运行测试与构建
go test ./...
go build ./...

# 5. 提交
git add idl/ kitex_gen/ && git commit -m "feat(idl): add XXX"
```

### 服务布局速览

```
rpc/
	ticket/{main.go,impl/impl.go}
	kb/{main.go,impl/impl.go}
	ai/{main.go,impl/impl.go}
services/gateway/           # HTTP 入口（本地可直连内存实现，或转发 RPC）
cmd/rpc-probe/              # 内联启动多 RPC 的探针工具
idl/                        # Thrift 定义（单一事实来源）
kitex_gen/                  # 生成代码（禁止手改）
script/verify-gen.sh        # 漂移检测脚本
```

### 术语约定
- impl 包：纯业务实现，可被 gateway、probe、测试直接 import。
- regen：通过 kitex 工具再生产所有服务代码。
- 漂移（drift）：IDL 与 kitex_gen 目录内容不一致。

### 常用指南入口
- API 契约：`./guides/api-contracts.md`
- KB 搜索原理：`./guides/kb-search-principles.md`
- Elasticsearch 运维：`./guides/kb-es-ops.md`
- 测试策略：`./guides/testing-strategy.md`
- Runbook：`./guides/runbook.md`

### 会话记录规范（可选）
- 命名：`YYYY-MM-DD-session-XX.md`
- 摘要：记录决策、风险、后续工作。
- 详细：保留关键命令输出 / Diff / 性能数据。

### 后续改进想法
- 自动生成 API markdown（从 Thrift 或注释提取）
- 将 verify-gen 纳入 CI
- 补充端到端 benchmark（ticket 创建→搜索→embedding）
