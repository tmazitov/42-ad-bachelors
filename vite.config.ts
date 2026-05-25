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
    proxy: {
      '/api/42': {
        target: 'https://api.intra.42.fr',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api\/42/, ''),
      },
    },
  },
})
