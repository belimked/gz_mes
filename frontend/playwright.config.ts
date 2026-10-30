import { defineConfig, devices } from '@playwright/test'

// E2E_BASE_URL set  -> run against a real stack (docker compose).
// E2E_BASE_URL unset -> build the demo bundle and test it with vite preview.
const baseURL = process.env.E2E_BASE_URL

export default defineConfig({
  testDir: 'e2e',
  timeout: 30_000,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? [['github'], ['html', { open: 'never' }]] : 'list',
  use: {
    baseURL: baseURL ?? 'http://127.0.0.1:14173',
    trace: 'retain-on-failure',
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
  webServer: baseURL
    ? undefined
    : {
        command: 'npm run build:demo && npx vite preview --host 127.0.0.1 --port 14173 --strictPort',
        url: 'http://127.0.0.1:14173',
        reuseExistingServer: !process.env.CI,
        timeout: 180_000,
      },
})
