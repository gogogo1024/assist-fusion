# AssistFusion Web (Vite + React)

## 开发

1) 启动后端（gateway）

2) 使用 pnpm 安装依赖并启动前端：

```sh
cd web
pnpm install
pnpm dev
```

默认端口 5173，已配置代理到 `http://localhost:8081` 的 /v1、/ready、/metrics。

## 构建

```sh
pnpm build
pnpm preview  # 4173 预览
```

构建产物位于 `web/dist/`。请提交 `pnpm-lock.yaml` 上锁，保证团队一致。

## 复杂业务流 / 状态机（XState）

- 位置：`src/machines/ticketMachine.ts`
- 内容：
	- Ticket 状态：created/assigned/in_progress/waiting/escalated/resolved/closed/canceled
	- 事件：ASSIGN/START/WAIT/ESCALATE/RESOLVE/REOPEN/CLOSE/CANCEL
	- 辅助：`canTransition(status, action)` 用于前端校验按钮可用性，内置终态限制（终态禁止 escalate/start/wait 等）。
- 使用示例：
	- 坐席页（`src/ui/Agent.tsx`）中，根据当前选中工单状态动态禁用按钮；若触发非法操作，给予提示并阻止请求。

## 国际化（i18n）

- 依赖：`i18next`、`react-i18next`、`i18next-browser-languagedetector`
- 初始化：`src/i18n/index.ts`，默认中文（zh-CN），支持 `en`，自动探测浏览器语言并缓存到 localStorage，可通过 URL `?lang=zh-CN|en` 切换
- 资源：
	- 中文：`src/locales/zh-CN/common.json`
	- 英文：`src/locales/en/common.json`
- 使用：在组件内 `const { t } = useTranslation()`，用 `t('nav.supervisor')` 等 key 获取文案
- 切换：页面顶部右侧的语言下拉框，或 URL 参数 `lang`
