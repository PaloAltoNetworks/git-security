// https://nuxt.com/docs/api/configuration/nuxt-config
import { createProxyServer } from "httpxy"
import { IncomingMessage } from "http"
import internal from "stream"

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
    'dayjs-nuxt',
    'nuxt-icon',
    'nuxt-lodash'
  ],
  dayjs: {
    plugins: ['duration', 'relativeTime'],
  },
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
  hooks: {
    listen(server) {
      const proxy = createProxyServer({
        ws: true,
        secure: false,
        changeOrigin: true,
        target: { host: "localhost", port: 8080 }
      })

      const proxyFn = (req: IncomingMessage, socket: internal.Duplex, head: Buffer) => {
        if (req.url && req.url.startsWith("/ws")) {
          // @ts-ignore
          proxy.ws(req, socket, head)
        }
      }
      server.on("upgrade", proxyFn)
      console.log("websocket dev proxy started")
    }
  },
  ui: {
    icons: ['fa6-solid']
  },
  routeRules: {
    '/settings': { redirect: '/settings/columns' },
  },
})
