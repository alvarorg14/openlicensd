import { expect, test } from '@playwright/test'

import {
  apiCreateUser,
  apiLogin,
  createProduct,
  login,
  testUser,
  uniqueSuffix
} from './helpers'

test('admin sees audit events after a mutation', async ({ page }) => {
  const suffix = uniqueSuffix()
  const productName = `E2E Audit Product ${suffix}`
  const productCode = `e2e-audit-${suffix}`

  await login(page)
  await createProduct(page, productName, productCode)

  await page.goto('/audit-log')
  await expect(page.getByRole('heading', { name: 'Audit Log' })).toBeVisible()
  await expect(page.locator('tbody tr').filter({ hasText: productName })).toBeVisible()
})

test('operator is redirected from audit log', async ({ page, request }) => {
  const suffix = uniqueSuffix()
  const operator = testUser('operator', suffix)

  await apiLogin(request)
  await apiCreateUser(request, operator)
  await login(page, operator.email, operator.password)

  await page.goto('/audit-log')
  await expect(page).toHaveURL(/\/licenses$/)
})
