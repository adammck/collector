/// <reference types="vitest" />
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html'],
      exclude: [
        'node_modules/',
        'src/test/',
        '**/*.d.ts',
      ],
    },
  },
  server: {
    proxy: {
      '/data.json': 'http://localhost:8000',
      '/submit': 'http://localhost:8000',
      '/defer': 'http://localhost:8000',
      '/queue': 'http://localhost:8000',
    }
  }
})