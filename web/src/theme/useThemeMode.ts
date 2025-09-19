import { useCallback, useEffect, useState } from 'react'
import { refreshArcoTheme } from './arco-theme'

export type ThemeMode = 'minimal' | 'vivid'

const STORAGE_KEY = 'app-theme-mode'

function applyBodyClass(mode: ThemeMode){
  document.body.classList.remove('minimal','vivid')
  document.body.classList.add(mode)
}

export function useThemeMode(): [ThemeMode, (m: ThemeMode)=>void, ()=>void] {
  const [mode, setMode] = useState<ThemeMode>(()=>{
    const saved = (typeof window !== 'undefined' && localStorage.getItem(STORAGE_KEY)) as ThemeMode | null
    return saved || 'minimal'
  })

  const update = useCallback((m: ThemeMode)=>{
    setMode(m)
    if (typeof window !== 'undefined') {
      localStorage.setItem(STORAGE_KEY, m)
    }
    applyBodyClass(m)
    // 重新读取 body class 下的 CSS 变量同步给 Arco theme
    refreshArcoTheme()
  },[])

  const toggle = useCallback(()=>{
    update(mode === 'minimal' ? 'vivid' : 'minimal')
  },[mode, update])

  useEffect(()=>{
    applyBodyClass(mode)
    refreshArcoTheme()
  },[]) // 初始化一次

  return [mode, update, toggle]
}
