# 运行手册（Runbook）

日期：2025-09-12

## 先决条件
- 安装 mise
- Go >= 1.22
- macOS 需安装 Xcode Command Line Tools（修复 dyld 问题）

## 常用命令
```sh
# 依赖整理
go mod tidy

# 启动各 RPC 服务（开发）
make run-ticket  # :8201 (TICKET_RPC_ADDR 可覆盖)
make run-kb      # :8202
make run-ai      # :8203

# 启动 gateway（本地直连模式）
HTTP_ADDR=:8081 go run ./services/gateway

# 切换 gateway 为 RPC 模式（需要上述 RPC 已起）
FEATURE_RPC=true HTTP_ADDR=:8081 go run ./services/gateway

# 再生成并校验 Kitex 代码
make regen
make verify-gen

# 测试 / Lint
go test ./...
golangci-lint run
```

## 健康检查
- 访问 http://localhost:8081/health 或根路径（若已实现）
- 查看服务日志，确认中间件输出请求轨迹

## 故障排查
- dyld: missing LC_UUID → 安装/修复 CLT、清理 go 缓存
- 端口占用 → 修改 HTTP_ADDR 或释放端口
- 依赖冲突 → go mod tidy、固定版本

## Airflow（可选）
- 将 `airflow/dags/kb_ingest_dag.py` 放至 Airflow 的 dags 目录
- 配置环境变量：KB_DATA_DIR、KB_SVC、AI_SVC
- 在 WebUI 手动触发运行
