export default defineNuxtConfig({
  ssr: false,
  devtools: { enabled: true },
  modules: ['@nuxt/ui', '@nuxt/eslint'],
  css: ['~/assets/css/main.css'],
  app: {
    head: {
      title: 'openlicensd',
      meta: [
        { name: 'description', content: 'Open source license server' }
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
  compatibilityDate: '2025-01-01'
})
