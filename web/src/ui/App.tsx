import React, { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import Agent from './Agent'
import Supervisor from './Supervisor'
import KnowledgeBase from './KnowledgeBase'
import Diag from './Diag'
import { Tabs, TabsList, TabsTrigger } from '../components/ui/tabs'

type Tab = 'agent'|'supervisor'|'kb'|'diag'

export default function App(){
  const { t, i18n } = useTranslation()
  const [tab, setTab] = useState<Tab>('agent')
  const tabs: {key:Tab,label:string}[] = useMemo(() => ([
    { key:'agent', label: t('nav.agent') },
    { key:'supervisor', label: t('nav.supervisor') },
    { key:'kb', label: t('nav.kb') },
    { key:'diag', label: t('nav.diag') },
  ]), [t])
  return (
    <>
      <header className="sticky top-0 z-10 border-b border-border bg-[hsl(230_55%_8%/0.9)] backdrop-blur supports-[backdrop-filter]:bg-[hsl(230_55%_8%/0.6)]">
        <div className="container py-3 flex items-center gap-2">
          <Tabs className="max-w-full" value={tab} onValueChange={(v)=> setTab(v as Tab)}>
            <TabsList>
              {tabs.map(tb => (
                <TabsTrigger key={tb.key} value={tb.key} active={tab===tb.key} onClick={() => setTab(tb.key)}>
                  {tb.label}
                </TabsTrigger>
              ))}
            </TabsList>
          </Tabs>
          <div className="ml-auto flex items-center gap-2">
            <select
              className="input h-9"
              value={i18n.language}
              onChange={(e)=> i18n.changeLanguage(e.target.value)}
              aria-label="language switcher"
            >
              <option value="zh-CN">中文</option>
              <option value="en">English</option>
            </select>
          </div>
        </div>
      </header>
      <div className="container py-4">
        {tab==='agent' && <Agent />}
        {tab==='supervisor' && <Supervisor />}
  {tab==='kb' && <KnowledgeBase />}
        {tab==='diag' && <Diag />}
      </div>
    </>
  )
}
