import react from '@vitejs/plugin-react'
import { defineConfig, loadEnv } from 'vite'

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const target = env.API_PROXY_TARGET || 'http://localhost:8080'
  const proxy = Object.fromEntries(
    ['/api', '/livez', '/readyz'].map((path) => [path, { target, changeOrigin: true }]),
  )

  return {
    plugins: [react()],
    server: { proxy },
    preview: { proxy },
  }
})
