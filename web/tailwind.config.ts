import type { Config } from 'tailwindcss'

export default {
  darkMode: ['class'],
  content: [
    './index.html',
    './src/**/*.{ts,tsx}',
  ],
  theme: {
    extend: {
      colors: {
        background: 'hsl(230 60% 6%)',
        foreground: 'hsl(220 20% 95%)',
        card: {
          DEFAULT: 'hsl(230 55% 8%)',
          foreground: 'hsl(220 20% 95%)'
        },
        primary: {
          DEFAULT: 'hsl(230 90% 60%)',
          foreground: '#fff'
        },
        muted: {
          DEFAULT: 'hsl(230 35% 16%)',
          foreground: 'hsl(220 12% 70%)'
        },
        border: 'hsl(230 35% 18%)'
      },
      borderRadius: {
        lg: '10px',
        md: '8px',
        sm: '6px'
      },
      boxShadow: {
        glow: '0 0 0 1px hsl(230 90% 60% / 0.3), 0 8px 30px -12px hsl(230 90% 60% / 0.35)'
      }
    }
  },
  plugins: [],
} satisfies Config
