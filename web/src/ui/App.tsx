import React, { useMemo, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
// 按需懒加载各页面，减少首屏体积 (C: 加 chunk name 注释帮助构建产物可读)
const Agent = React.lazy(() => import(/* webpackChunkName: "tab-agent" */ './Agent'))
const Supervisor = React.lazy(() => import(/* webpackChunkName: "tab-supervisor" */ './Supervisor'))
const KnowledgeBase = React.lazy(() => import(/* webpackChunkName: "tab-kb" */ './KnowledgeBase'))
const Diag = React.lazy(() => import(/* webpackChunkName: "tab-diag" */ './Diag'))
import { Tabs, Select, Message, Spin } from '@arco-design/web-react'
import { Button } from '../components/ui/button'

type Tab = 'agent'|'supervisor'|'kb'|'diag'

export default function App(){
  const { t, i18n } = useTranslation()
  // 初始激活：支持通过 URL hash (#agent 等) 直接定位
  const initial = (typeof window !== 'undefined' && window.location.hash.replace('#','')) as Tab || 'agent'
  const [tab, setTab] = React.useState<Tab>(initial)

  // 同步 hash（切换后更新地址，不产生历史记录堆栈）
  useEffect(() => {
    if (typeof window !== 'undefined') {
      const newHash = '#' + tab
      if (window.location.hash !== newHash) history.replaceState(null,'',newHash)
    }
  }, [tab])
  // 模拟各模块待办/统计数，可与真实数据对接
  // 动态计数：放在 state，模拟实时变化
  const [counts, setCounts] = React.useState<Partial<Record<Tab, number>>>(
    { agent: 3, supervisor: 1, kb: 0, diag: 2 }
  )
  // 记录上一轮值，用于后续可能的对比（E: 接入真实 API 后可删除）
  const prevCounts = React.useRef(counts)
  // 临时增量动画数据结构：key -> delta (B)
  const [flashes, setFlashes] = React.useState<Partial<Record<Tab, number>>>()
  // 提供外部（真实数据 / WebSocket / 轮询）调用的更新函数 (E)
  const updateCounts = React.useCallback((updater: Partial<Record<Tab, number>> | ((prev: Partial<Record<Tab, number>>) => Partial<Record<Tab, number>>)) => {
    setCounts(prev => {
      const next = typeof updater === 'function' ? (updater as any)(prev) : { ...prev, ...updater }
      // 计算差异触发动画
      const diff: Partial<Record<Tab, number>> = {}
      ;(['agent','supervisor','kb','diag'] as Tab[]).forEach(k => {
        const a = prev[k] || 0
        const b = next[k] || 0
        if (a !== b) diff[k] = b - a
      })
      if (Object.keys(diff).length) {
        setFlashes(diff)
        scheduleFlashCleanup(diff)
      }
      prevCounts.current = next
      return next
    })
  }, [])

  // 暴露到 window 方便临时调试（生产可移除）
  useEffect(() => {
    if (typeof window !== 'undefined') {
      ;(window as any).__updateTabCounts = updateCounts
    }
  }, [updateCounts])
  // B: 浮动增量动画清理封装，避免深层嵌套
  function scheduleFlashCleanup(diff: Partial<Record<Tab, number>>) {
    setTimeout(() => {
      setFlashes(curr => {
        if (!curr) return curr
        const clone = { ...curr }
        for (const dk of Object.keys(diff) as Tab[]) delete clone[dk]
        return Object.keys(clone).length ? clone : undefined
      })
    }, 1200)
  }
  const tabs: {key:Tab; label:string; content:React.ReactNode; count?:number}[] = useMemo(() => ([
    { key:'agent', label: t('nav.agent'), content: <Agent />, count: counts.agent },
    { key:'supervisor', label: t('nav.supervisor'), content: <Supervisor />, count: counts.supervisor },
    { key:'kb', label: t('nav.kb'), content: <KnowledgeBase />, count: counts.kb },
    { key:'diag', label: t('nav.diag'), content: <Diag />, count: counts.diag },
  ]), [t, counts])

  // 键盘快捷键：Ctrl/Alt + ←/→ 切换；数字 1-4 快速直达
  useEffect(() => {
    const order = tabs.map(t => t.key)
    const handler = (e: KeyboardEvent) => {
      const activeEl = document.activeElement as HTMLElement | null
      // 避免在输入框/可编辑元素里触发
      if (activeEl && (activeEl.tagName === 'INPUT' || activeEl.tagName === 'SELECT' || activeEl.isContentEditable)) return
      if ((e.ctrlKey || e.metaKey || e.altKey) && (e.key === 'ArrowRight' || e.key === 'ArrowLeft')) {
        e.preventDefault()
        const idx = order.indexOf(tab)
        if (idx !== -1) {
          const next = e.key === 'ArrowRight' ? (idx + 1) % order.length : (idx - 1 + order.length) % order.length
          setTab(order[next])
        }
      } else if (!e.ctrlKey && !e.metaKey && !e.altKey && /^[1-4]$/.test(e.key)) {
        const num = parseInt(e.key, 10) - 1
  if (order[num]) setTab(order[num])
      }
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [tab, tabs])
  return (
    <>
      <header className="sticky top-0 z-10 border-b border-border bg-[hsl(230_55%_8%/0.9)] backdrop-blur supports-[backdrop-filter]:bg-[hsl(230_55%_8%/0.6)]">
        <div className="container py-3 flex items-center gap-2">
          <Tabs
            className="max-w-full advanced-tabs"
            activeTab={tab}
            onChange={(k)=> setTab(k as Tab)}
            type="line"
            lazyload
            animation
            extra={(
              <div className="flex items-center gap-2 pl-4 border-l border-[var(--c-border,#2a323d)]">
                <Button size="small" type="secondary" disabled>{t('actions.vivid')}</Button>
                <Select
                  size="small"
                  style={{ width: 120 }}
                  value={i18n.language}
                  onChange={(val)=> {
                    i18n.changeLanguage(val as string)
                    Message.success(val === 'zh-CN' ? '已切换为中文' : 'Switched to English')
                  }}
                  aria-label="language switcher"
                  triggerProps={{ autoAlignPopupWidth:false }}
                  dropdownMenuClassName="lang-select-dropdown"
                >
                  <Select.Option value="zh-CN">中文</Select.Option>
                  <Select.Option value="en">English</Select.Option>
                </Select>
              </div>
            )}
          >
            {tabs.map(tb => {
              const delta = flashes?.[tb.key]
              const n = tb.count ?? 0
              const title = (
                <span style={{ display:'inline-flex', alignItems:'center', gap:6, position:'relative' }}>
                  {tb.label}
                  {n > 0 && (
                    <span className="tab-badge-lite" data-variant={n > 20 ? 'warn' : 'normal'}>
                      <span className="dot" />{n}
                    </span>
                  )}
                  {delta && delta !== 0 && (
                    <span className={"tab-badge-float " + (delta > 0 ? 'up' : 'down')}>
                      {delta > 0 ? '+' + delta : delta}
                    </span>
                  )}
                </span>
              )
              return (
                <Tabs.TabPane key={tb.key} title={title}>
                  <React.Suspense fallback={<div className="py-10 flex justify-center"><Spin /></div>}>
                    {tb.content}
                  </React.Suspense>
                </Tabs.TabPane>
              )
            })}
          </Tabs>
          {/* 右侧冗余语言切换 & 按钮已移除，集中到 Tabs extra 中 */}
        </div>
      </header>
      {/* 内容已内移 TabPane；保留容器做额外内边距控制 */}
      <div className="container py-4" />
    </>
  )
}
