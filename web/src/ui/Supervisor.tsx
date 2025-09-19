import React, { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { get, put } from '../utils/api'
import { Card } from '@arco-design/web-react'
import { Statistic, Tag, Input, Table as ArcoTable } from '@arco-design/web-react'
import { statusToVariant, statKeyToVariant, variantBarClass, variantLeftBarClass, variantTintClass, variantToTagColor, type Variant } from '../lib/status'
import { Button } from '../components/ui/button'
import { Skeleton } from '../components/ui/skeleton'
import { EmptyState } from '../components/ui/empty-state'
import { cn } from '../lib/utils'

export default function Supervisor(){
  const { t } = useTranslation()
  // 主题暂未在迁移后的按钮/表格里使用，可后续若加 dark 模式再恢复 useTheme()
  const [stats, setStats] = useState<any>({})
  const [unassigned, setUnassigned] = useState<any[]>([])
  const [overdue, setOverdue] = useState<any[]>([])
  const [loading, setLoading] = useState<boolean>(false)
  const [refreshing, setRefreshing] = useState<boolean>(false)
  // Variant type imported from status.ts
  const labelStatus = (s: string) => t(`stats.${String(s).toLowerCase()}`) || s
  const toVariant = statusToVariant
  const tagColor = variantToTagColor
  const statVariant = statKeyToVariant
  const barClass = variantBarClass

  // 模拟环比/同比变化：简单基于 key 做伪随机演示
  function mockDelta(key: string){
    const seed = Array.from(key).reduce((a,c)=>a + c.charCodeAt(0),0)
    const rand = (n:number)=> (Math.sin(seed + n)*10000)%1
    const base = rand(1)
    let trend: 'up'|'down'|'flat' = 'flat'
    if (base > 0.66) trend = 'up'; else if (base < 0.33) trend = 'down'
    const wow = ( (rand(2)*8).toFixed(1) + '%')
    const yoy = ( (rand(3)*15).toFixed(1) + '%')
    let value: string
    if (trend === 'up') value = '+' + wow
    else if (trend === 'down') value = '-' + wow
    else value = '0%'
    return { trend, wow, yoy, value }
  }
  const rowBarClass = variantLeftBarClass
  const rowTintClass = variantTintClass

  const UNASSIGNED_STATES = new Set(['created','waiting'])
  async function load(debug = false){
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
        // 未分配：限定状态集合（避免把已被分配后又清空 assignee 的奇异数据混进来）
        if (!t.assignee && UNASSIGNED_STATES.has(st)) ua.push(t)
        if (t.due_at && t.due_at < now && !term.has(st)) od.push(t)
      }
    }
    // 最新优先：假设有 created_at 时间戳（秒），若不存在则不排序
    ua.sort((a,b)=> (b.created_at||0)-(a.created_at||0))
    setStats(s)
    setUnassigned(ua.slice(0,50))
    setOverdue(od.slice(0,50))
    setLoading(false)
    if (debug) {
      // eslint-disable-next-line no-console
      console.log('[Supervisor debug] rawTickets=', list)
      // eslint-disable-next-line no-console
      console.log('[Supervisor debug] unassigned(after filter)=', ua.map(t=>({id:t.id,status:t.status,assignee:t.assignee,created_at:t.created_at})))
    }
  }
  useEffect(()=>{ load() }, [])

  async function assign(id:string, assignee:string){
    await put(`/v1/tickets/${id}/assign`, { assignee })
    load()
  }
  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
  <Button type='primary' onClick={async ()=>{ setRefreshing(true); try { await load() } finally { setRefreshing(false) }}} loading={refreshing}>
          {refreshing && (
            <svg className="mr-1 h-4 w-4 animate-spin text-current" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none" />
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v4A4 4 0 008 12H4z" />
            </svg>
          )}
          {t('actions.refresh')}
    </Button>
  <Button size="small" type='outline' onClick={()=>load(true)}>{t('actions.debug')||'Debug'}</Button>
      </div>
      <div className="text-xs text-muted-foreground">{t('guide.supervisor.hint')}</div>
      <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-3">
        {['total','created','assigned','in_progress','waiting','escalated','resolved','closed','canceled'].map(k => {
          const delta = mockDelta(k)
          let arrow = '→'
          if (delta.trend === 'up') arrow = '↑'
          else if (delta.trend === 'down') arrow = '↓'
          return (
            <Card key={k} className="stat-card stat-panel">
              <div className={cn('h-1 rounded mb-2', barClass(statVariant(k)))} />
              <div className="text-xs text-muted-foreground whitespace-nowrap leading-none tracking-wide stat-label">{t(`stats.${k}`)}</div>
              {loading ? (
                <Skeleton className="h-7 w-16 mt-2" />
              ) : (
                <div>
                  <Statistic
                    value={stats[k]||0}
                    style={{ fontSize: '1.5rem', letterSpacing: '.5px', fontVariantNumeric:'tabular-nums', fontWeight:600, lineHeight: '1.1' }}
                    suffix={<span className="text-xs text-muted-foreground ml-1">{delta.value}</span>}
                  />
                  <div className="mt-1 text-[10px] text-muted-foreground tracking-wide">{arrow} WoW {delta.wow} · YoY {delta.yoy}</div>
                </div>
              )}
            </Card>
          )
        })}
      </div>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
  <Card title={t('supervisor.unassigned')} className="p-4">
            {loading ? (
              <div className="space-y-2">
                {['a','b','c','d','e'].map(id => (<Skeleton key={`ua-${id}`} className="h-10 w-full" />))}
              </div>
            ) : (
            <ArcoTable
              size="small"
              pagination={false}
              rowKey="id"
              columns={[
                {
                  title: t('fields.id'),
                  dataIndex: 'id',
                  width: 70,
                  render: (value: any) => <code className="font-mono text-xs">{value}</code>
                },
                {
                  title: t('fields.title'),
                  dataIndex: 'title',
                  ellipsis: true,
                  render: (value: any, record: any) => (
                    <span
                      className="block font-medium text-foreground/90 leading-snug overflow-hidden text-ellipsis [display:-webkit-box] [-webkit-line-clamp:2] [-webkit-box-orient:vertical]"
                      title={value}
                    >
                      {value}
                    </span>
                  )
                },
                {
                  title: t('fields.status'),
                  dataIndex: 'status',
                  width: 90,
                  render: (_: any, record: any) => {
                    const v = toVariant(record.status)
                    return <Tag size="small" color={tagColor(v)}>{labelStatus(record.status)}</Tag>
                  }
                },
                {
                  title: t('fields.dueAt'),
                  dataIndex: 'due_at',
                  width: 110,
                  render: (val: any) => <span className="text-xs text-muted-foreground">{val || '-'}</span>
                },
                {
                  title: t('columns.operations'),
                  width: 160,
                  render: (_: any, record: any) => (
                    <div className="flex items-center gap-2">
                      <Input placeholder={t('fields.assignee')} id={`assignee-${record.id}`} style={{height:32, width:112}} />
                      <Button size="small" type='primary' onClick={()=>assign(record.id, (document.getElementById(`assignee-${record.id}`) as HTMLInputElement)?.value||'')}>{t('actions.assign')}</Button>
                    </div>
                  )
                }
              ]}
              data={unassigned}
              noDataElement={<EmptyState title={t('common.empty')} description={t('guide.empty.tickets')} />}
              rowClassName={(record: any)=>{
                const v = toVariant(record.status)
                const bar = rowBarClass(v)
                const tint = rowTintClass(v)
                return `left-bar ${bar} ${tint}`
              }}
            />
            )}
          </Card>
  <Card title={t('supervisor.overdue')} className="p-4">
            {loading ? (
              <div className="space-y-2">
                {['a','b','c','d','e'].map(id => (<Skeleton key={`od-${id}`} className="h-10 w-full" />))}
              </div>
            ) : (
            <ArcoTable
              size="small"
              pagination={false}
              rowKey="id"
              columns={[
                {
                  title: t('fields.id'),
                  dataIndex: 'id',
                  width: 70,
                  render: (value: any) => <code className="font-mono text-xs">{value}</code>
                },
                {
                  title: t('fields.title'),
                  dataIndex: 'title',
                  ellipsis: true,
                  render: (value: any) => (
                    <span
                      className="block font-medium text-foreground/90 leading-snug overflow-hidden text-ellipsis [display:-webkit-box] [-webkit-line-clamp:2] [-webkit-box-orient:vertical]"
                      title={value}
                    >
                      {value}
                    </span>
                  )
                },
                {
                  title: t('fields.status'),
                  dataIndex: 'status',
                  width: 90,
                  render: (_: any, record: any) => {
                    const v = toVariant(record.status)
                    return <Tag size="small" color={tagColor(v)}>{labelStatus(record.status)}</Tag>
                  }
                },
                {
                  title: t('fields.dueAt'),
                  dataIndex: 'due_at',
                  width: 110,
                  render: (val: any) => <span className="text-xs">{val}</span>
                },
                {
                  title: t('columns.operations'),
                  width: 140,
                  render: () => <span className="opacity-60 text-xs italic">--</span>
                }
              ]}
              data={overdue}
              noDataElement={<EmptyState title={t('common.empty')} description={t('guide.empty.tickets')} />}
              rowClassName={(record: any)=>{
                const v = toVariant(record.status)
                const bar = rowBarClass(v)
                const tint = rowTintClass(v)
                return `left-bar ${bar} ${tint}`
              }}
            />
            )}
          </Card>
      </div>
      <div className="mt-4 flex flex-wrap gap-2 text-xs">
        {([
          { k:'success', label:'stats.resolved', color:'var(--color-success)' },
          { k:'warning', label:'stats.escalated', color:'var(--color-warning)' },
          { k:'danger', label:'stats.canceled', color:'var(--color-danger)' },
          { k:'info', label:'stats.in_progress', color:'var(--color-info)' },
          { k:'neutral', label:'stats.closed', color:'var(--color-neutral)' }
        ] as const).map(item => (
          <Tag
            key={item.k}
            color={item.color}
            size="small"
            style={{ display:'inline-flex', alignItems:'center' }}
          >
            {t(item.label as any)}
          </Tag>
        ))}
      </div>
    </div>
  )
}
