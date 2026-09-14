import { expect, test } from '@playwright/test'

import {
  apiCreateUser,
  apiLogin,
  createLicense,
  createPolicy,
  createProduct,
  login,
  testUser,
  uniqueSuffix
} from './helpers'

test.describe('RBAC', () => {
  test('viewer cannot mutate or access admin pages', async ({ page, request }) => {
    const suffix = uniqueSuffix()
    const viewer = testUser('viewer', suffix)

    await apiLogin(request)
    await apiCreateUser(request, viewer)
    await login(page, viewer.email, viewer.password)

    await page.goto('/licenses')
    await expect(page.getByRole('button', { name: 'Create license' })).toHaveCount(0)

    await page.goto('/products')
    await expect(page.getByRole('button', { name: 'Create product' })).toHaveCount(0)

    await page.goto('/users')
    await expect(page).toHaveURL(/\/licenses$/)
  })

  test('operator can manage licenses', async ({ page, request }) => {
    const suffix = uniqueSuffix()
    const operator = testUser('operator', suffix)
    const productName = `E2E Operator Product ${suffix}`
    const productCode = `e2e-op-${suffix}`
    const policyName = `E2E Operator Policy ${suffix}`
    const licenseLabel = `E2E Operator License ${suffix}`

    await apiLogin(request)
    await apiCreateUser(request, operator)
    await login(page, operator.email, operator.password)

    await createProduct(page, productName, productCode)
    await createPolicy(page, productName, policyName)
    await createLicense(page, productName, policyName, licenseLabel)

    await expect(page.getByRole('dialog')).toHaveCount(0)
    await expect(page.locator('tbody').getByText(licenseLabel)).toBeVisible()
  })
})
