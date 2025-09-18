import React, { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { get } from '../utils/api'
import { Card, CardContent, CardTitle } from '../components/ui/card'
import { Button } from '../components/ui/button'

export default function Diag(){
  const { t } = useTranslation()
  const [out, setOut] = useState<any>('')
  return (
    <div className="space-y-4">
      <Card>
        <CardTitle className="mb-2">{t('nav.diag')}</CardTitle>
        <CardContent className="flex flex-wrap items-center gap-2">
          <Button onClick={async()=>{ const r=await get('/ready'); setOut(r) }}>{t('actions.checkReady')}</Button>
          <Button variant="outline" onClick={async()=>{ const r=await fetch('/metrics/domain'); setOut(await r.text()) }}>{t('actions.viewMetrics')}</Button>
        </CardContent>
      </Card>
      <pre>{typeof out==='string'? out: JSON.stringify(out,null,2)}</pre>
    </div>
  )
}
