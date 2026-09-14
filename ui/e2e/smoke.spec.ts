import { expect, test } from '@playwright/test'

import { createProductPolicyLicense, login } from './helpers'

test('login, create product, policy, license, and validate key', async ({ page, request }) => {
  await login(page)
  const { productCode, licenseKey } = await createProductPolicyLicense(page)

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
