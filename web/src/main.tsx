import React from 'react'
import { createRoot } from 'react-dom/client'
import App from './ui/App'
import './index.css'
import './i18n'

const el = document.getElementById('root')!
createRoot(el).render(<App />)
