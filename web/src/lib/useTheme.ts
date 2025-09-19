import { useCallback, useEffect, useState } from 'react'

export type UITheme = 'minimal' | 'vivid'
const STORAGE_KEY = 'ui-theme'
const LEGACY_VIVID_KEY = 'vivid-mode'

function applyTheme(theme: UITheme) {
  if (typeof document === 'undefined') return
  document.body.classList.remove('minimal', 'vivid')
  document.body.classList.add(theme)
}

export function useTheme(): [UITheme, (t: UITheme)=>void, ()=>void] {
  const [theme, setTheme] = useState<UITheme>(() => {
    if (typeof window === 'undefined') return 'minimal'
    // migration: legacy vivid-mode key
    if (localStorage.getItem(LEGACY_VIVID_KEY)) {
      localStorage.removeItem(LEGACY_VIVID_KEY)
      localStorage.setItem(STORAGE_KEY, 'vivid')
      return 'vivid'
    }
    const stored = localStorage.getItem(STORAGE_KEY) as UITheme | null
    return stored === 'vivid' ? 'vivid' : 'minimal'
  })

  useEffect(() => { applyTheme(theme); localStorage.setItem(STORAGE_KEY, theme) }, [theme])

  const set = useCallback((t: UITheme) => setTheme(t), [])
  const toggle = useCallback(() => setTheme(t => t === 'vivid' ? 'minimal' : 'vivid'), [])
  return [theme, set, toggle]
}

export function useIsVivid() {
  const [theme] = useTheme()
  return theme === 'vivid'
}
