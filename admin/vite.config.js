import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig({
  plugins: [vue()],
  base: '/admin/',
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src')
    }
  },
  server: {
    port: 3001,
    proxy: {
      '/api': {
        target: 'http://localhost:8380',
        changeOrigin: true
      },
      '/file': {
        target: 'http://localhost:8380',
        changeOrigin: true
      }
    }
  },
  build: {
    outDir: '../server/static/embed/admin',
    emptyOutDir: true
  }
})
