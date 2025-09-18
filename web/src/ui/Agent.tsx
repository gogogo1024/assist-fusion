import React, { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { get, put, post } from '../utils/api'
import { canTransition } from '../machines/ticketMachine'
import { Card, CardContent, CardTitle } from '../components/ui/card'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'
import { Badge } from '../components/ui/badge'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow, TableCaption } from '../components/ui/table'
import { Skeleton } from '../components/ui/skeleton'
import { EmptyState } from '../components/ui/empty-state'

type Variant = 'default'|'success'|'warning'|'danger'|'neutral'|'info'

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
}

const ActionBtn: React.FC<Readonly<ActionBtnProps>> = ({ name, label, variant = 'default', disabled, title, loading, onClick }) => (
  <Button
    size="sm"
    variant={variant}
    onClick={onClick}
    disabled={disabled}
    title={disabled ? (title || undefined) : undefined}
    aria-busy={loading}
    aria-label={typeof label === 'string' ? label : name}
  >
    {loading && <Spinner />}
    {label}
  </Button>
)

export default function Agent(){
  const { t } = useTranslation()
  const [list, setList] = useState<any[]>([])
  const [loading, setLoading] = useState<boolean>(false)
  const [title, setTitle] = useState(t('agent.demoTitle'))
  const [desc, setDesc] = useState(t('agent.demoDesc'))
  const [id, setId] = useState('')
  const [note, setNote] = useState('')
  const [out, setOut] = useState<any>(null)
  const [actionLoading, setActionLoading] = useState<string|null>(null)
  const toVariant = (s: string): 'default'|'success'|'warning'|'danger'|'neutral'|'info' => {
    const v = String(s||'').toLowerCase()
    switch (v) {
      case 'resolved': return 'success'
      case 'escalated': return 'warning'
      case 'closed': return 'neutral'
      case 'canceled': return 'danger'
      case 'waiting':
      case 'in_progress': return 'info'
      default: return 'default'
    }
  }

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

  

  return (
    <div className="space-y-4">
      <Card>
        <CardTitle className="mb-2">{t('agent.heading')}</CardTitle>
        <CardContent className="space-y-3">
          <div className="text-xs text-muted-foreground">
            {id ? t('guide.agent.current', { id }) : t('guide.agent.hint')}
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <Input value={title} onChange={e=>setTitle(e.target.value)} placeholder={t('fields.title')} className="h-9 w-56" />
            <Input value={desc} onChange={e=>setDesc(e.target.value)} placeholder={t('fields.desc')} className="h-9 w-72" />
            <Button onClick={create}>{t('actions.create')}</Button>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <Input value={id} onChange={e=>setId(e.target.value)} placeholder={t('fields.ticketId')} className="h-9 w-48" />
            <Input value={note} onChange={e=>setNote(e.target.value)} placeholder={t('fields.note')} className="h-9 w-72" />
            {(() => {
              const cur = list.find(t=>t.id===id)
              const s = (cur?.status||'created')
              const names = ['assign','start','wait','escalate','resolve','close','cancel','reopen'] as const
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
                    onClick={()=>act(name)}
                  />
                )
              })
              return (<div className="flex flex-wrap items-center gap-2">{nodes}</div>)
            })()}
          </div>
        </CardContent>
      </Card>
      <Card>
        <CardContent>
          {loading ? (
            <div className="space-y-2">
              <Skeleton className="h-6 w-64" />
              {['a','b','c','d','e'].map(id => (
                <Skeleton key={`row-skel-${id}`} className="h-10 w-full" />
              ))}
            </div>
          ) : (
          <Table>
            {loading && <TableCaption>{t('common.loading')}</TableCaption>}
            <TableHeader>
              <TableRow>
                <TableHead>{t('fields.id')}</TableHead>
                <TableHead>{t('fields.title')}</TableHead>
                <TableHead>{t('fields.status')}</TableHead>
                <TableHead>{t('columns.operations')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {!loading && list.length===0 && (
                <TableRow>
                  <TableCell colSpan={4}>
                    <EmptyState title={t('common.empty')} description={t('guide.empty.tickets')} />
                  </TableCell>
                </TableRow>
              )}
              {list.map((row:any)=> (
                <TableRow key={row.id} onClick={()=>setId(row.id)} style={{cursor:'pointer'}}>
                  <TableCell><code>{row.id}</code></TableCell>
                  <TableCell>{row.title}</TableCell>
                  <TableCell>
                    <Badge variant={toVariant(row.status)}>
                      {t(`stats.${String(row.status).toLowerCase()}`)}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <Button size="sm" variant={id===row.id? 'default':'outline'} onClick={(e)=>{ e.stopPropagation(); setId(row.id) }}>{id===row.id? t('columns.operations'): t('actions.assign')}</Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
          )}
        </CardContent>
      </Card>
      <pre>{out ? JSON.stringify(out, null, 2) : ''}</pre>
    </div>
  )
}
