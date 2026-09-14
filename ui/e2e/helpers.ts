import { expect, type APIRequestContext, type Locator, type Page } from '@playwright/test'

export const ADMIN_EMAIL = 'admin@example.com'
export const ADMIN_PASSWORD = 'admin'

const CSRF_COOKIE_NAME = 'openlicensd_csrf'
const CSRF_HEADER_NAME = 'X-CSRF-Token'

export interface LicenseFixture {
  productName: string
  productCode: string
  policyName: string
  licenseLabel: string
  licenseKey: string
}

export interface TestUserCredentials {
  name: string
  email: string
  password: string
  role: 'viewer' | 'operator' | 'admin'
}

/**
 * Select an option from a Nuxt UI USelectMenu using its placeholder text.
 */
export async function selectMenuOption(
  page: Page,
  placeholder: string,
  optionText: string,
  scope?: Locator
): Promise<void> {
  const root = scope ?? page
  const trigger = root
    .getByRole('combobox')
    .filter({ hasText: placeholder })
    .or(root.getByPlaceholder(placeholder))
    .or(root.locator('button').filter({ hasText: placeholder }))
    .first()

  await trigger.click()
  const option = page.getByRole('option', { name: optionText, exact: true })
  await option.click()
  await expect(option).toBeHidden()
}

/**
 * Fill a Vue-controlled input after ensuring it has focus.
 */
export async function fillInput(input: Locator, value: string): Promise<void> {
  await input.click()
  await input.fill(value)
  await expect(input).toHaveValue(value)
}

/**
 * Submit the primary action in a dialog.
 */
export async function submitDialog(dialog: Locator, actionName: string): Promise<void> {
  await dialog.getByRole('button', { name: actionName, exact: true }).click()
}

export async function login(
  page: Page,
  email: string = ADMIN_EMAIL,
  password: string = ADMIN_PASSWORD
): Promise<void> {
  await page.goto('/login')
  await page.getByLabel('Email', { exact: true }).fill(email)
  await page.getByLabel('Password', { exact: true }).fill(password)
  await page.getByRole('button', { name: 'Sign in', exact: true }).click()
  await expect(page).toHaveURL(/\/licenses$/)
}

async function getCsrfToken(request: APIRequestContext): Promise<string> {
  const state = await request.storageState()
  const csrf = state.cookies.find((cookie) => cookie.name === CSRF_COOKIE_NAME)?.value
  if (!csrf) {
    throw new Error('missing CSRF cookie after login')
  }
  return csrf
}

export async function apiLogin(
  request: APIRequestContext,
  email: string = ADMIN_EMAIL,
  password: string = ADMIN_PASSWORD
): Promise<void> {
  const response = await request.post('/api/v1/auth/login', {
    data: { email, password }
  })
  expect(response.ok()).toBeTruthy()
}

export async function apiCreateUser(
  request: APIRequestContext,
  user: TestUserCredentials
): Promise<void> {
  const csrf = await getCsrfToken(request)
  const response = await request.post('/api/v1/users', {
    data: {
      email: user.email,
      name: user.name,
      password: user.password,
      role: user.role
    },
    headers: {
      [CSRF_HEADER_NAME]: csrf
    }
  })
  expect(response.ok()).toBeTruthy()
}

export function uniqueSuffix(): number {
  return Date.now()
}

export function testUser(role: 'viewer' | 'operator', suffix: number = uniqueSuffix()): TestUserCredentials {
  return {
    name: `E2E ${role} ${suffix}`,
    email: `e2e-${role}-${suffix}@example.com`,
    password: `${role}-pass-123`,
    role
  }
}

export async function createProduct(
  page: Page,
  productName: string,
  productCode: string
): Promise<void> {
  await page.goto('/products')
  await page.getByRole('button', { name: 'Create product' }).click()
  const productDialog = page.getByRole('dialog')
  await fillInput(productDialog.getByRole('textbox', { name: /^Name/ }), productName)
  await fillInput(productDialog.getByRole('textbox', { name: /^Code/ }), productCode)
  await submitDialog(productDialog, 'Create')
  await expect(page.getByText(productName)).toBeVisible()
}

export async function createPolicy(
  page: Page,
  productName: string,
  policyName: string
): Promise<void> {
  await page.goto('/policies')
  await page.getByRole('button', { name: 'Create policy' }).click()
  const policyDialog = page.getByRole('dialog')
  await selectMenuOption(page, 'Select a product', productName, policyDialog)
  await fillInput(policyDialog.getByRole('textbox', { name: /^Name/ }), policyName)
  await submitDialog(policyDialog, 'Create')
  await expect(page.getByText(policyName)).toBeVisible()
}

export async function createLicense(
  page: Page,
  productName: string,
  policyName: string,
  licenseLabel: string
): Promise<string> {
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
  return licenseKey
}

export async function createProductPolicyLicense(
  page: Page,
  suffix: number = uniqueSuffix()
): Promise<LicenseFixture> {
  const productName = `E2E Product ${suffix}`
  const productCode = `e2e-${suffix}`
  const policyName = `E2E Policy ${suffix}`
  const licenseLabel = `E2E License ${suffix}`

  await createProduct(page, productName, productCode)
  await createPolicy(page, productName, policyName)
  const licenseKey = await createLicense(page, productName, policyName, licenseLabel)

  return {
    productName,
    productCode,
    policyName,
    licenseLabel,
    licenseKey
  }
}

export async function openRowActions(page: Page, rowLabel: string): Promise<void> {
  const row = page.locator('tbody tr').filter({ hasText: rowLabel })
  await row.locator('button').last().click()
}

export function licenseRow(page: Page, rowLabel: string) {
  return page.locator('tbody tr').filter({ hasText: rowLabel })
}

export async function confirmModal(page: Page, confirmLabel: string): Promise<void> {
  const dialog = page.getByRole('dialog')
  await expect(dialog).toBeVisible()
  await dialog.getByRole('button', { name: confirmLabel, exact: true }).click()
  await expect(dialog).toBeHidden()
}
