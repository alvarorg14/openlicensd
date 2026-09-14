import { expect, test } from '@playwright/test'

import {
  confirmModal,
  createProductPolicyLicense,
  licenseRow,
  login,
  openRowActions,
  uniqueSuffix
} from './helpers'

test('admin revokes and deletes a license', async ({ page }) => {
  const suffix = uniqueSuffix()

  await login(page)
  const { licenseLabel } = await createProductPolicyLicense(page, suffix)

  await page.goto('/licenses')
  await expect(page.getByText(licenseLabel)).toBeVisible()

  await openRowActions(page, licenseLabel)
  await page.getByRole('menuitem', { name: 'Revoke' }).click()
  await confirmModal(page, 'Revoke')

  await expect(licenseRow(page, licenseLabel)).toContainText('Revoked')

  await openRowActions(page, licenseLabel)
  await page.getByRole('menuitem', { name: 'Delete' }).click()
  await confirmModal(page, 'Delete')

  await expect(page.locator('tbody').getByText(licenseLabel)).toHaveCount(0)
})
