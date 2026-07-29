import path from 'path'
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { visualizer } from 'rollup-plugin-visualizer'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [
    react(),
    // Bundle analyzer — run `ANALYZE=1 npm run build` to open the report.
    ...(process.env.ANALYZE
      ? [
          visualizer({
            filename: 'dist-analyze/stats.html',
            open: true,
            gzipSize: true,
            brotliSize: true,
          }),
        ]
      : []),
  ],
  build: {
    outDir: '../internal/transports/grpc/dist',
    emptyOutDir: true,
    target: ['chrome110', 'edge110', 'firefox115', 'safari16'],
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes('node_modules')) {
            // Generated proto code is large, stable, and imported app-wide —
            // keeping it out of the entry chunk lets it cache independently.
            if (id.includes('/src/gen/')) return 'app-proto'
            return undefined
          }
          if (/node_modules\/(react|react-dom|react-router|react-router-dom|scheduler)\//.test(id)) return 'vendor-react'
          if (id.includes('@tanstack/react-query')) return 'vendor-query'
          if (/node_modules\/(@connectrpc|@bufbuild)\//.test(id)) return 'vendor-grpc'
          if (/node_modules\/(date-fns|diff|zod|zustand|clsx|tailwind-merge)\//.test(id)) return 'vendor-utils'
          if (/node_modules\/(cmdk|sonner|react-hotkeys-hook|@tanstack\/react-virtual)\//.test(id)) return 'vendor-ui'
          return undefined
        },
      },
    },
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 5173,
    proxy: {
      // Proxy gRPC-web requests to the backend during development
      '/natscope.': {
        target: 'http://localhost:4280',
        changeOrigin: true,
      },
    },
  },
})
