import { defineConfig, devices } from '@playwright/test'

export default defineConfig({
  testDir: './e2e',
  use: {
    ...devices['Desktop Chrome'],
    baseURL: 'http://127.0.0.1:4173/openlicensd/',
  },
  webServer: {
    command: 'npm run docs:preview -- --host 127.0.0.1 --port 4173',
    url: 'http://127.0.0.1:4173/openlicensd/',
    reuseExistingServer: !process.env.CI,
  },
})
