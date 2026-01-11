import { defineVitestConfig } from '@nuxt/test-utils/config'

export default defineVitestConfig({
  test: {
    environment: 'nuxt',
    environmentOptions: {
      nuxt: {
        domEnvironment: 'happy-dom',
      },
    },
    // Test file patterns
    include: ['**/*.{test,spec}.{js,ts,vue}'],
    exclude: ['node_modules', '.nuxt', 'dist'],
    // Coverage configuration
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html'],
      include: ['components/**/*.vue', 'composables/**/*.ts', 'pages/**/*.vue'],
      exclude: ['node_modules', '.nuxt', 'dist'],
    },
    // Global test timeout
    testTimeout: 10000,
    // Reporter
    reporters: ['default'],
  },
})
