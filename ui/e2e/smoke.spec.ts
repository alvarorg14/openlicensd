import { expect, test } from '@playwright/test'

import { fillInput, selectMenuOption, submitDialog } from './helpers'

const ADMIN_EMAIL = 'admin@example.com'
const ADMIN_PASSWORD = 'admin'

test('login, create product, policy, license, and validate key', async ({ page, request }) => {
  const suffix = Date.now()
  const productName = `E2E Product ${suffix}`
  const productCode = `e2e-${suffix}`
  const policyName = `E2E Policy ${suffix}`
  const licenseLabel = `E2E License ${suffix}`

  await page.goto('/login')
  await page.getByLabel('Email', { exact: true }).fill(ADMIN_EMAIL)
  await page.getByLabel('Password', { exact: true }).fill(ADMIN_PASSWORD)
  await page.getByRole('button', { name: 'Sign in', exact: true }).click()
  await expect(page).toHaveURL(/\/licenses$/)

  await page.goto('/products')
  await page.getByRole('button', { name: 'Create product' }).click()
  const productDialog = page.getByRole('dialog')
  await fillInput(productDialog.getByRole('textbox', { name: /^Name/ }), productName)
  await fillInput(productDialog.getByRole('textbox', { name: /^Code/ }), productCode)
  await submitDialog(productDialog, 'Create')
  await expect(page.getByText(productName)).toBeVisible()

  await page.goto('/policies')
  await page.getByRole('button', { name: 'Create policy' }).click()
  const policyDialog = page.getByRole('dialog')
  await selectMenuOption(page, 'Select a product', productName, policyDialog)
  await fillInput(policyDialog.getByRole('textbox', { name: /^Name/ }), policyName)
  await submitDialog(policyDialog, 'Create')
  await expect(page.getByText(policyName)).toBeVisible()

  await page.goto('/licenses')
  await page.getByRole('button', { name: 'Create license' }).click()
  const licenseDialog = page.getByRole('dialog')
  await fillInput(licenseDialog.getByRole('textbox', { name: /^Label/ }), licenseLabel)
  await selectMenuOption(page, 'Select a product', productName, licenseDialog)
  await selectMenuOption(page, 'Select a policy', policyName, licenseDialog)
  await submitDialog(licenseDialog, 'Create')

  const licenseKeyInput = page.getByLabel('License key')
  await expect(licenseKeyInput).toBeVisible()
  const licenseKey = await licenseKeyInput.inputValue()
  expect(licenseKey.length).toBeGreaterThan(0)
  await page.getByRole('button', { name: 'Done' }).click()

  const validateResponse = await request.post('/api/v1/validate', {
    data: {
      key: licenseKey,
      product: productCode
    }
  })
  expect(validateResponse.ok()).toBeTruthy()
  const body = await validateResponse.json()
  expect(body.valid).toBe(true)
})
