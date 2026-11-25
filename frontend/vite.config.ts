import { fileURLToPath, URL } from 'node:url'
import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vitest/config'
import type { ProxyOptions } from 'vite'

const API_TARGET = 'http://localhost:8080'

/**
 * The API is a separate process (`cd backend && go run ./cmd/server`). Without
 * it, every proxied call fails with ECONNREFUSED and Vite prints a stack trace
 * per request. Warn once and answer with the API's error envelope instead.
 */
function apiProxyErrors(label: string): ProxyOptions['configure'] {
  let warned = false

  return (proxy) => {
    proxy.on('error', (error, _request, response) => {
      if (!warned) {
        warned = true
        console.warn(
          `[vite] ${label} proxy cannot reach ${API_TARGET} (${error.message}). ` +
            'Start the API with: cd backend && go run ./cmd/server',
        )
      }

      if ('writeHead' in response && !response.headersSent) {
        response.writeHead(503, { 'content-type': 'application/json' })
        response.end(
          JSON.stringify({
            error: {
              code: 'api_unavailable',
              message: 'The API is not running. Start it with: cd backend && go run ./cmd/server',
              details: {},
            },
          }),
        )
      }
    })

    // The API came back: allow a fresh warning if it drops again.
    proxy.on('proxyRes', () => {
      warned = false
    })
  }
}

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    proxy: {
      '/api': {
        target: API_TARGET,
        changeOrigin: true,
        configure: apiProxyErrors('/api'),
      },
      '/healthz': {
        target: API_TARGET,
        changeOrigin: true,
        configure: apiProxyErrors('/healthz'),
      },
      '/readyz': {
        target: API_TARGET,
        changeOrigin: true,
        configure: apiProxyErrors('/readyz'),
      },
      '/ws': {
        target: API_TARGET.replace(/^http/, 'ws'),
        ws: true,
        configure: apiProxyErrors('/ws'),
      },
    },
  },
  build: {
    outDir: 'dist',
  },
  test: {
    environment: 'jsdom',
    include: ['src/**/*.{test,spec}.ts'],
    restoreMocks: true,
  },
})
