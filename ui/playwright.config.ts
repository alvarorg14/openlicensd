import { defineConfig, devices } from '@playwright/test'

const baseURL = process.env.PLAYWRIGHT_BASE_URL ?? 'http://127.0.0.1:8080'

export default defineConfig({
  testDir: './e2e',
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  workers: 1,
  reporter: process.env.CI ? [['html', { open: 'never' }], ['list']] : 'list',
  use: {
    baseURL,
    trace: process.env.CI ? 'on-first-retry' : 'off'
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] }
    }
  ],
  webServer: {
    command: 'bin/openlicensd',
    url: `${baseURL}/healthz`,
    reuseExistingServer: !process.env.CI,
    cwd: '..',
    env: {
      ...process.env,
      OPENLICENSD_ADDR: ':8080',
      OPENLICENSD_COOKIE_SECURE: 'false',
      OPENLICENSD_METRICS_ENABLED: 'false'
    } as Record<string, string>
  }
})
