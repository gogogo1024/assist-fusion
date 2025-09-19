import React, { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { get, post, put, del } from '../utils/api'
import { Card, Input } from '@arco-design/web-react'
import { Button } from '../components/ui/button'

export default function KnowledgeBase(){
  const { t } = useTranslation()
  const [title, setTitle] = useState('FAQ')
  const [content, setContent] = useState('示例内容')
  const [id, setId] = useState('')
  const [q, setQ] = useState('')
  const [out, setOut] = useState<any>('')
  return (
    <div className="space-y-4">
      <Card title={t('nav.kb')} className="p-4">
        <div className="space-y-3">
          <div className="flex flex-wrap items-center gap-2">
            <Input value={title} onChange={(v)=>setTitle(v)} placeholder={t('fields.title')} className="h-9 w-56" />
            <Input value={content} onChange={(v)=>setContent(v)} placeholder={t('fields.content')} className="h-9 w-80" />
            <Button onClick={async()=>{ const r=await post('/v1/docs',{title,content}); setOut(r); setId(r?.id||'') }}>{t('actions.add')}</Button>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <Input value={id} onChange={(v)=>setId(v)} placeholder={t('fields.docId')} className="h-9 w-48" />
            <Button onClick={async()=>{ const r=await put(`/v1/docs/${id}`,{title,content}); setOut(r) }}>{t('actions.update')}</Button>
            <Button type="outline" onClick={async()=>{ const r=await del(`/v1/docs/${id}`); setOut(r) }}>{t('actions.delete')}</Button>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <Input value={q} onChange={(v)=>setQ(v)} placeholder={t('fields.query')} className="h-9 w-72" />
            <Button onClick={async()=>{ const r=await get(`/v1/search?q=${encodeURIComponent(q)}&limit=10`); setOut(r) }}>{t('actions.search')}</Button>
            <Button type="text" onClick={async()=>{ const r=await get('/v1/kb/info'); setOut(r) }}>{t('actions.backendInfo')}</Button>
          </div>
        </div>
      </Card>
      <pre>{out? JSON.stringify(out,null,2):''}</pre>
    </div>
  )
}
