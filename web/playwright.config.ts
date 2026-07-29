import { defineConfig, devices } from '@playwright/test'

/**
 * E2E tests run against the Vite dev server (:5173) proxying gRPC-web calls
 * to the local natscope backend (:4280), which must be running and connected
 * to a local NATS server (nats://localhost:4222, saved connection "local").
 *
 * Workers = 1: tests share the backend (publish history, templates store)
 * and the browser's localStorage-backed draft store.
 */
export default defineConfig({
  testDir: './e2e',
  timeout: 30_000,
  workers: 1,
  fullyParallel: false,
  retries: 0,
  reporter: [['list']],
  use: {
    baseURL: 'http://localhost:5173',
    trace: 'retain-on-failure',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
  webServer: {
    command: 'npm run dev -- --port 5173 --strictPort',
    url: 'http://localhost:5173',
    reuseExistingServer: true,
    timeout: 60_000,
  },
})
