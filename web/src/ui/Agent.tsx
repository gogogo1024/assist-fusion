import React, { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { get, put, post } from '../utils/api'
import { canTransition } from '../machines/ticketMachine'
import { Card, Input, Tag, Table as ArcoTable } from '@arco-design/web-react'
import { statusToVariant, variantLeftBarClass, variantTintClass, variantToTagColor, type Variant } from '../lib/status'
import { Button } from '../components/ui/button'
import { Skeleton } from '../components/ui/skeleton'
import { EmptyState } from '../components/ui/empty-state'
import { useTheme } from '../lib/useTheme'

// Variant type imported from status.ts

const Spinner: React.FC = () => (
  <svg className="mr-1 h-4 w-4 animate-spin text-current" viewBox="0 0 24 24">
    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none" />
    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v4A4 4 0 008 12H4z" />
  </svg>
)

interface ActionBtnProps {
  readonly name: string
  readonly label: React.ReactNode
  readonly variant?: Variant
  readonly disabled?: boolean
  readonly title?: string
  readonly loading?: boolean
  readonly onClick: () => void
  readonly className?: string
}

const ActionBtn: React.FC<Readonly<ActionBtnProps>> = ({ name, label, variant = 'default', disabled, title, loading, onClick, className }) => {
  // 简单 variant 到 Arco type 映射（默认 primary，危险 danger -> primary + 自定义色后续可扩展）
  let type: any = 'primary'
  if (variant === 'warning' || variant === 'neutral' || variant === 'info') type = 'outline'
  if (variant === 'default') type = 'secondary'
  if (variant === 'danger') type = 'primary'
  return (
    <Button
      size="small"
      type={type}
      onClick={onClick}
      disabled={disabled}
      title={disabled ? (title || undefined) : undefined}
      aria-busy={loading}
      aria-label={typeof label === 'string' ? label : name}
      className={className}
    >
      {loading && <Spinner />}
      {label}
    </Button>
  )
}

export default function Agent(){
  const { t } = useTranslation()
  const [theme] = useTheme()
  // vivid 模式已移除
  const [list, setList] = useState<any[]>([])
  const [loading, setLoading] = useState<boolean>(false)
  const [title, setTitle] = useState(t('agent.demoTitle'))
  const [desc, setDesc] = useState(t('agent.demoDesc'))
  const [id, setId] = useState('')
  const [note, setNote] = useState('')
  const [out, setOut] = useState<any>(null)
  const [actionLoading, setActionLoading] = useState<string|null>(null)
  const toVariant = statusToVariant

  async function load(){
    setLoading(true)
    try {
      const r = await get('/v1/tickets')
      if (Array.isArray(r)) setList(r)
    } finally {
      setLoading(false)
    }
  }
  useEffect(() => { load() }, [])

  async function create(){
    const r = await post('/v1/tickets', { title, desc })
    setOut(r)
    if (r?.id) setId(r.id)
    load()
  }
  async function act(name:string){
    if (!id) return alert(t('agent.selectTicketFirst'))
    const cur = list.find(t=>t.id===id)
  const s = (cur?.status||'created')
    if (!canTransition(s, name as any)){
      alert(t('agent.illegalTransition', { state: s, action: name }))
      return
    }
    setActionLoading(name)
    try {
      const r = await put(`/v1/tickets/${id}/${name}`, { note })
      setOut(r)
      await load()
    } finally {
      setActionLoading(null)
    }
  }

  const actionVariant = (name: string): Variant => {
    switch (name) {
      case 'resolve': return 'success'
      case 'escalate': return 'warning'
      case 'wait': return 'warning'
      case 'close': return 'danger'
      case 'cancel': return 'danger'
      case 'assign': return 'info'
      case 'start': return 'info'
      case 'reopen': return 'info'
      default: return 'default'
    }
  }

  const rowBarClass = variantLeftBarClass
  const rowTintClass = variantTintClass
  const tagColor = variantToTagColor

  

  return (
    <div className="space-y-4">
  <Card title={t('agent.heading')} className="p-4">
    <div className="space-y-3">
          <div className="text-xs text-muted-foreground">
            {id ? t('guide.agent.current', { id }) : t('guide.agent.hint')}
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <Input value={title} onChange={(v)=>setTitle(v)} placeholder={t('fields.title')} className="h-9 w-56" />
            <Input value={desc} onChange={(v)=>setDesc(v)} placeholder={t('fields.desc')} className="h-9 w-72" />
            <Button type="primary" onClick={create}>{t('actions.create')}</Button>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <Input value={id} onChange={(v)=>setId(v)} placeholder={t('fields.ticketId')} className="h-9 w-48" />
            <Input value={note} onChange={(v)=>setNote(v)} placeholder={t('fields.note')} className="h-9 w-72" />
            {(() => {
              const cur = list.find(t=>t.id===id)
              const s = (cur?.status||'created')
              const names = ['assign','start','wait','escalate','resolve','close','cancel','reopen'] as const
                const vivid = theme === 'vivid'
                const nodes = names.map((name) => {
                let disabled = false
                let title: string | undefined
                if (!id) {
                  disabled = true
                  title = t('agent.selectTicketFirst')
                } else if (!canTransition(s, name as any)) {
                  disabled = true
                  title = t('agent.illegalTransition', { state: s, action: name })
                } else if (actionLoading) {
                  disabled = true
                }
                const loadingNow = actionLoading === name
                return (
                    <ActionBtn
                    key={name}
                    name={name}
                    label={t(`actions.${name}`)}
                    variant={actionVariant(name)}
                    disabled={disabled}
                    title={title}
                    loading={loadingNow}
                    className={vivid ? 'neon-border' : undefined}
                    onClick={()=>act(name)}
                  />
                )
              })
              return (<div className="flex flex-wrap items-center gap-2">{nodes}</div>)
            })()}
          </div>
        </div>
      </Card>
  <Card className="p-4">
          {loading ? (
            <div className="space-y-2">
              <Skeleton className="h-6 w-64" />
              {['a','b','c','d','e'].map(id => (
                <Skeleton key={`row-skel-${id}`} className="h-10 w-full" />
              ))}
            </div>
          ) : (
          <ArcoTable
            size="small"
            pagination={false}
            rowKey="id"
            columns={[
              { title: t('fields.id'), dataIndex: 'id', width: 70, render: (v:any)=><code>{v}</code> },
              { title: t('fields.title'), dataIndex: 'title', ellipsis: true },
              { title: t('fields.status'), dataIndex: 'status', width: 90, render: (_:any, record:any)=> {
                  const variant = toVariant(record.status)
                  return <Tag size="small" color={tagColor(variant)}>{t(`stats.${String(record.status).toLowerCase()}`)}</Tag>
                }
              },
              { title: t('columns.operations'), width: 150, render: (_:any, record:any)=> {
                  const active = id===record.id
                  return <Button size="small" type={active? 'primary':'outline'} onClick={()=>setId(record.id)}>{active? t('columns.operations'): t('actions.assign')}</Button>
                }
              }
            ]}
            data={list}
            noDataElement={<EmptyState title={t('common.empty')} description={t('guide.empty.tickets')} />}
            rowClassName={(record:any)=>{
              const variant = toVariant(record.status)
              const bar = rowBarClass(variant)
              const tint = rowTintClass(variant)
              const active = id===record.id
              const cls = ['left-bar', bar, 'transition-colors', 'hover:bg-muted/60']
              if (active) cls.push('is-active', tint)
              return cls.join(' ')
            }}
            onRow={(record:any)=>({ onClick: ()=> setId(record.id), style:{cursor:'pointer'} })}
          />
          )}
        </Card>
      <pre>{out ? JSON.stringify(out, null, 2) : ''}</pre>
    </div>
  )
}
