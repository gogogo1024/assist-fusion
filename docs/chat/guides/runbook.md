# 运行手册（Runbook）

日期：2025-09-12

## 先决条件
- 安装 mise
- Go >= 1.22
- macOS 需安装 Xcode Command Line Tools（修复 dyld 问题）

## 常用命令
```sh
# 整理依赖
docs/chat$ mise run bootstrap

# 运行工单服务（:8081）
docs/chat$ mise run run-ticket

#（创建服务后解注释）运行 kb-svc（:8082）与 ai-svc（:8083）
# docs/chat$ mise run run-kb
# docs/chat$ mise run run-ai

# 运行测试
docs/chat$ mise run test

# 静态检查（如已安装）
docs/chat$ mise run lint
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
