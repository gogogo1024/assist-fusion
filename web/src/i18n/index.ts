import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import LanguageDetector from 'i18next-browser-languagedetector'

// Import locale resources
import zhCN from '../locales/zh-CN/common.json'
import en from '../locales/en/common.json'

void i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources: {
      'zh-CN': { translation: zhCN },
      zh: { translation: zhCN },
      en: { translation: en },
    },
    fallbackLng: 'zh-CN',
    supportedLngs: ['zh-CN', 'zh', 'en'],
    interpolation: { escapeValue: false },
    detection: {
      // Order: querystring -> localStorage -> navigator
      order: ['querystring', 'localStorage', 'navigator'],
      caches: ['localStorage'],
      lookupQuerystring: 'lang',
    },
  })

export default i18n
