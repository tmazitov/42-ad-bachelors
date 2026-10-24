import { fileURLToPath, URL } from 'node:url'

import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [tailwindcss(), vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    // The app and the API share one origin, as in production behind the
    // reverse proxy, so the session cookie is first-party.
    proxy: {
      '/api': { target: 'http://localhost:8080' },
    },
  },
})
