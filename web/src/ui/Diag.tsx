import React, { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { get } from '../utils/api'
import { Card } from '@arco-design/web-react'
import { Button } from '../components/ui/button'

export default function Diag(){
  const { t } = useTranslation()
  const [out, setOut] = useState<any>('')
  return (
    <div className="space-y-4">
      <Card title={t('nav.diag')} className="p-4">
        <div className="flex flex-wrap items-center gap-2">
          <Button onClick={async()=>{ const r=await get('/ready'); setOut(r) }}>{t('actions.checkReady')}</Button>
          <Button type="outline" onClick={async()=>{ const r=await fetch('/metrics/domain'); setOut(await r.text()) }}>{t('actions.viewMetrics')}</Button>
        </div>
      </Card>
      <pre>{typeof out==='string'? out: JSON.stringify(out,null,2)}</pre>
    </div>
  )
}
