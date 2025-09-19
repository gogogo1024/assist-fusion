// Arco Design 主题 token 映射：从现有 CSS 变量 / 语义色过渡
// 可根据需要扩展更多 token（参见 Arco Theme 文档）


// 读取当前文档根变量的工具（运行时获取 CSS Custom Properties）
function readCssVar(name: string, fallback: string): string {
  if (typeof window === 'undefined') return fallback
  const v = getComputedStyle(document.documentElement).getPropertyValue(name).trim()
  return v || fallback
}

// 基础映射：primary / success / warning / danger / info
const primary = () => readCssVar('--c-brand-primary', '#165DFF')
const success = () => readCssVar('--color-success', '#00B42A')
const warning = () => readCssVar('--color-warning', '#FF7D00')
const danger = () => readCssVar('--color-danger', '#F53F3F')
const info = () => readCssVar('--color-info', '#168CFF') // 读取 info 颜色

// Arco ConfigProvider theme 属性接受局部 token 覆盖，这里按需提供函数（延迟读取）
// 使用宽松类型，避免依赖内部未导出类型；后续可根据官方声明再收紧。
export const themeTokens: Partial<Record<string, any>> = {
  colorPrimary: primary(),
  colorSuccess: success(),
  colorWarning: warning(),
  colorDanger: danger(),
  colorLink: primary(),
  // 自定义扩展：供局部样式消费
  colorInfo: info()
}

export function refreshArcoTheme(){
  // 若运行时主题切换（minimal / vivid）后需要重新同步，可重新赋值并强制刷新
  Object.assign(themeTokens, {
    colorPrimary: primary(),
    colorSuccess: success(),
    colorWarning: warning(),
    colorDanger: danger(),
    colorLink: primary(),
    colorInfo: info()
  })
}

export const colorInfo = () => info() // 导出 colorInfo 访问器
