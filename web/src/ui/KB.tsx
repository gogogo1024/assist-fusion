import React, { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { get, post, put, del } from '../utils/api'

export default function KB(){
  const { t } = useTranslation()
  const [title, setTitle] = useState('FAQ')
  const [content, setContent] = useState('示例内容')
  const [id, setId] = useState('')
  const [q, setQ] = useState('')
  const [out, setOut] = useState<any>('')
  return (
    <div className="card">
      <h2>{t('nav.kb')}</h2>
      <div className="row">
        <input value={title} onChange={e=>setTitle(e.target.value)} placeholder={t('fields.title')} />
        <input value={content} onChange={e=>setContent(e.target.value)} placeholder={t('fields.content')} />
        <button onClick={async()=>{ const r=await post('/v1/docs',{title,content}); setOut(r); setId(r?.id||'') }}>{t('actions.add')}</button>
      </div>
      <div className="row">
        <input value={id} onChange={e=>setId(e.target.value)} placeholder={t('fields.docId')} />
        <button onClick={async()=>{ const r=await put(`/v1/docs/${id}`,{title,content}); setOut(r) }}>{t('actions.update')}</button>
        <button onClick={async()=>{ const r=await del(`/v1/docs/${id}`); setOut(r) }}>{t('actions.delete')}</button>
      </div>
      <div className="row">
        <input value={q} onChange={e=>setQ(e.target.value)} placeholder={t('fields.query')} />
        <button onClick={async()=>{ const r=await get(`/v1/search?q=${encodeURIComponent(q)}&limit=10`); setOut(r) }}>{t('actions.search')}</button>
        <button onClick={async()=>{ const r=await get('/v1/kb/info'); setOut(r) }}>{t('actions.backendInfo')}</button>
      </div>
      <pre>{out? JSON.stringify(out,null,2):''}</pre>
    </div>
  )
}
