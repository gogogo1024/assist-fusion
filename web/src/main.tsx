import React from 'react'
import { createRoot } from 'react-dom/client'
import App from './ui/App'
import './index.css'
import '@arco-design/web-react/dist/css/arco.css'
import './i18n'
import { ConfigProvider } from '@arco-design/web-react'
import zhCN from '@arco-design/web-react/es/locale/zh-CN'
import { themeTokens } from './theme/arco-theme'

const el = document.getElementById('root')!
createRoot(el).render(
	<ConfigProvider locale={zhCN} theme={themeTokens}>
		<App />
	</ConfigProvider>
)
