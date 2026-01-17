// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  compatibilityDate: '2024-11-01',
  devtools: { enabled: true },

  modules: ['@nuxtjs/i18n', '@nuxt/eslint'],

  i18n: {
    locales: [
      { code: 'ja', name: '日本語', file: 'ja.json' },
      { code: 'en', name: 'English', file: 'en.json' }
    ],
    defaultLocale: 'ja',
    langDir: 'locales',
    strategy: 'no_prefix',
    detectBrowserLanguage: {
      useCookie: true,
      cookieKey: 'i18n_locale',
      fallbackLocale: 'ja'
    }
  },

  components: [
    {
      path: '~/components',
      pathPrefix: false,
    }
  ],

  runtimeConfig: {
    public: {
      apiBase: process.env.API_BASE_URL || 'http://localhost:8080/api/v1',
      keycloakUrl: process.env.KEYCLOAK_URL || 'http://localhost:8180',
      keycloakRealm: process.env.KEYCLOAK_REALM || 'ai-orchestration',
      keycloakClientId: process.env.KEYCLOAK_CLIENT_ID || 'frontend'
    }
  },

  app: {
    head: {
      title: 'AI Orchestration',
      meta: [
        { charset: 'utf-8' },
        { name: 'viewport', content: 'width=device-width, initial-scale=1' },
        { name: 'description', content: 'DAG Workflow Orchestration Platform' }
      ]
    }
  },

  css: ['~/assets/css/main.css'],

  typescript: {
    strict: true,
    typeCheck: true
  }
})
