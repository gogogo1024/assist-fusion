import React, { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { get, put } from '../utils/api'
import { Card, CardTitle, CardContent } from '../components/ui/card'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'
import { Badge } from '../components/ui/badge'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../components/ui/table'
import { Skeleton } from '../components/ui/skeleton'
import { EmptyState } from '../components/ui/empty-state'
import { cn } from '../lib/utils'

export default function Supervisor(){
  const { t } = useTranslation()
  const [stats, setStats] = useState<any>({})
  const [unassigned, setUnassigned] = useState<any[]>([])
  const [overdue, setOverdue] = useState<any[]>([])
  const [loading, setLoading] = useState<boolean>(false)
  const [refreshing, setRefreshing] = useState<boolean>(false)
  type Variant = 'default'|'success'|'warning'|'danger'|'neutral'|'info'
  const labelStatus = (s: string) => t(`stats.${String(s).toLowerCase()}`) || s
  const toVariant = (s: string): Variant => {
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
  const statVariant = (k: string): Variant => {
    switch (k) {
      case 'resolved': return 'success'
      case 'escalated': return 'warning'
      case 'waiting': return 'warning'
      case 'closed': return 'neutral'
      case 'canceled': return 'danger'
      case 'in_progress': return 'info'
      case 'assigned': return 'info'
      case 'created': return 'info'
      default: return 'default' // total and others
    }
  }
  const barClass = (v: Variant) => {
    switch (v) {
      case 'success': return 'bg-emerald-500'
      case 'warning': return 'bg-amber-500'
      case 'danger': return 'bg-rose-500'
      case 'neutral': return 'bg-muted'
      case 'info': return 'bg-sky-500'
      default: return 'bg-primary'
    }
  }

  async function load(){
    setLoading(true)
    const list = await get('/v1/tickets')
    const s: any = { total: Array.isArray(list)? list.length: 0 }
    const term = new Set(['resolved','closed','canceled'])
    const ua: any[] = []
    const od: any[] = []
    const now = Math.floor(Date.now()/1000)
    if (Array.isArray(list)){
      for (const t of list){
        const st = String(t.status||'').toLowerCase();
        s[st] = (s[st]||0)+1
        if (!t.assignee) ua.push(t)
        if (t.due_at && t.due_at < now && !term.has(st)) od.push(t)
      }
    }
    setStats(s)
    setUnassigned(ua.slice(0,50))
    setOverdue(od.slice(0,50))
    setLoading(false)
  }
  useEffect(()=>{ load() }, [])

  async function assign(id:string, assignee:string){
    await put(`/v1/tickets/${id}/assign`, { assignee })
    load()
  }
  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        <Button variant="info" onClick={async ()=>{ setRefreshing(true); try { await load() } finally { setRefreshing(false) }}} aria-busy={refreshing}>
          {refreshing && (
            <svg className="mr-1 h-4 w-4 animate-spin text-current" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none" />
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v4A4 4 0 008 12H4z" />
            </svg>
          )}
          {t('actions.refresh')}
        </Button>
      </div>
      <div className="text-xs text-muted-foreground">{t('guide.supervisor.hint')}</div>
      <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-3">
        {['total','created','assigned','in_progress','waiting','escalated','resolved','closed','canceled'].map(k => (
          <Card key={k} className="p-4">
            <div className={cn('h-1 rounded mb-2', barClass(statVariant(k)))} />
            <div className="text-xs text-muted-foreground">{t(`stats.${k}`)}</div>
            {loading ? (
              <Skeleton className="h-7 w-16 mt-2" />
            ) : (
              <div className="text-2xl font-bold">{stats[k]||0}</div>
            )}
          </Card>
        ))}
      </div>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <Card>
          <CardTitle className="mb-2">{t('supervisor.unassigned')}</CardTitle>
          <CardContent>
            {loading ? (
              <div className="space-y-2">
                {['a','b','c','d','e'].map(id => (<Skeleton key={`ua-${id}`} className="h-10 w-full" />))}
              </div>
            ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('fields.id')}</TableHead>
                  <TableHead>{t('fields.title')}</TableHead>
                  <TableHead>{t('fields.status')}</TableHead>
                  <TableHead>{t('columns.operations')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {!loading && unassigned.length===0 && (
                  <TableRow>
                    <TableCell colSpan={4}><EmptyState title={t('common.empty')} description={t('guide.empty.tickets')} /></TableCell>
                  </TableRow>
                )}
                {unassigned.map(row => (
                  <TableRow key={row.id}>
                    <TableCell><code>{row.id}</code></TableCell>
                    <TableCell>{row.title}</TableCell>
                    <TableCell><Badge variant={toVariant(row.status)}>{labelStatus(row.status)}</Badge></TableCell>
                    <TableCell className="space-x-2">
                      <Input placeholder={t('fields.assignee')} id={`assignee-${row.id}`} className="h-8 w-36" />
                      <Button size="sm" variant="success" onClick={()=>assign(row.id, (document.getElementById(`assignee-${row.id}`) as HTMLInputElement)?.value||'')}>{t('actions.assign')}</Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
            )}
          </CardContent>
        </Card>
        <Card>
          <CardTitle className="mb-2">{t('supervisor.overdue')}</CardTitle>
          <CardContent>
            {loading ? (
              <div className="space-y-2">
                {['a','b','c','d','e'].map(id => (<Skeleton key={`od-${id}`} className="h-10 w-full" />))}
              </div>
            ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('fields.id')}</TableHead>
                  <TableHead>{t('fields.title')}</TableHead>
                  <TableHead>{t('fields.status')}</TableHead>
                  <TableHead>{t('fields.dueAt')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {!loading && overdue.length===0 && (
                  <TableRow>
                    <TableCell colSpan={4}><EmptyState title={t('common.empty')} description={t('guide.empty.tickets')} /></TableCell>
                  </TableRow>
                )}
                {overdue.map(row => (
                  <TableRow key={row.id}>
                    <TableCell><code>{row.id}</code></TableCell>
                    <TableCell>{row.title}</TableCell>
                    <TableCell><Badge variant={toVariant(row.status)}>{labelStatus(row.status)}</Badge></TableCell>
                    <TableCell>{row.due_at}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
