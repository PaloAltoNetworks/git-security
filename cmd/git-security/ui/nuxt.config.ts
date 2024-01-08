// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  ssr: false,
  css: ['~/assets/scss/main.scss'],
  devtools: { enabled: true },
  typescript: {
    strict: true
  },
  app: {
    head: {
      title: 'Git Security'
    }
  },
  modules: [
    '@element-plus/nuxt',
    '@nuxt/ui',
    '@nuxtjs/color-mode',
    '@nuxtjs/google-fonts',
    '@vueuse/nuxt',
    'nuxt-icon',
    'nuxt-lodash'
  ],
  elementPlus: {
    importStyle: 'scss',
    icon: 'ElIcon'
  },
  googleFonts: {
    families: {
      "Montserrat": true
    }
  },
  nitro: {
    devProxy: {
      '/api/v1': {
        target: 'http://localhost:8080/api/v1',
        changeOrigin: true
      },
      '/logout': {
        target: 'http://localhost:8080/logout',
        changeOrigin: true
      }
    }
  },
  ui: {
    icons: ['fa6-solid']
  }
})
