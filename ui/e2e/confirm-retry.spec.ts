import { expect, test } from '@playwright/test'

import {
  createProductPolicyLicense,
  licenseRow,
  login,
  openRowActions,
  uniqueSuffix
} from './helpers'

test('confirm dialog retries after a failed mutation', async ({ page }) => {
  const suffix = uniqueSuffix()
  let revokeAttempts = 0

  await page.route('**/api/v1/licenses/*/revoke', async (route) => {
    revokeAttempts++
    if (revokeAttempts === 1) {
      await route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'simulated failure' })
      })
      return
    }
    await route.continue()
  })

  await login(page)
  const { licenseLabel } = await createProductPolicyLicense(page, suffix)

  await page.goto('/licenses')
  await expect(page.getByText(licenseLabel)).toBeVisible()

  await openRowActions(page, licenseLabel)
  await page.getByRole('menuitem', { name: 'Revoke' }).click()

  const dialog = page.getByRole('dialog')
  await expect(dialog).toBeVisible()

  await dialog.getByRole('button', { name: 'Revoke', exact: true }).click()
  await expect(dialog).toBeVisible()
  await expect(dialog.getByText('simulated failure')).toBeVisible()

  await dialog.getByRole('button', { name: 'Revoke', exact: true }).click()
  await expect(dialog).toBeHidden()

  await expect(licenseRow(page, licenseLabel)).toContainText('Revoked')
})
