import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/data.json': 'http://localhost:8000',
      '/submit': 'http://localhost:8000',
      '/defer': 'http://localhost:8000',
      '/queue': 'http://localhost:8000',
    }
  }
})
