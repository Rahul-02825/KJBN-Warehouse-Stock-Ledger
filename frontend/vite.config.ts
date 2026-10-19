import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// In development the browser talks only to Vite, which forwards API calls to the
// backend. One origin means no CORS setup is needed locally.
const apiTarget = process.env.API_PROXY_TARGET ?? 'http://localhost:8080'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    proxy: {
      '/api': apiTarget,
      '/healthz': apiTarget,
    },
  },
})
