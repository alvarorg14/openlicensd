import { APP_NAME } from './constants/app'

export default defineNuxtConfig({
  ssr: false,
  devtools: { enabled: true },
  modules: ['@nuxt/ui', '@nuxt/eslint'],
  css: ['~/assets/css/main.css'],
  fonts: {
    families: [
      { name: 'Space Grotesk', provider: 'none' }
    ]
  },
  app: {
    head: {
      title: APP_NAME,
      meta: [
        { name: 'description', content: 'Open source license server' }
      ],
      link: [
        { rel: 'icon', type: 'image/svg+xml', href: '/favicon.svg' }
      ]
    }
  },
  nitro: {
    output: {
      publicDir: '../server/internal/static/dist'
    }
  },
  devServer: {
    port: 3000
  },
  vite: {
    server: {
      proxy: {
        '/api': {
          target: 'http://localhost:8080',
          changeOrigin: true
        }
      }
    }
  },
  icon: {
    provider: 'none',
    clientBundle: {
      scan: {
        globInclude: ['**/*.{vue,jsx,tsx,md,mdc,mdx,yml,yaml,ts,js}']
      }
    }
  },
  compatibilityDate: '2025-01-01'
})
