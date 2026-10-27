import { expect, test, type Page } from '@playwright/test'

async function expectDiagrams(page: Page, count: number) {
  const blocks = page.locator('.language-mermaid')
  await expect(blocks).toHaveCount(count)
  for (let i = 0; i < count; i++) {
    const block = blocks.nth(i)
    await expect(async () => {
      await block.scrollIntoViewIfNeeded()
      await expect(block.locator('svg')).toBeVisible({ timeout: 5_000 })
      await expect(block.locator('.static-mermaid-container')).toBeVisible()
    }).toPass({ timeout: 30_000 })
  }
}

test('diagrams render across navigation and appearance changes', async ({ page }) => {
  test.setTimeout(90_000)
  const errors: string[] = []
  page.on('pageerror', error => errors.push(error.message))
  await page.goto('architecture')
  await expectDiagrams(page, 5)

  const firstDiagram = page.locator('.language-mermaid').first()
  await firstDiagram.scrollIntoViewIfNeeded()
  const oldSvgId = await firstDiagram.locator('svg').getAttribute('id')
  await page.locator('.VPSwitchAppearance:visible').first().click()
  await expect(page.locator('html')).toHaveClass(/dark/)
  await expect.poll(() => firstDiagram.locator('svg').getAttribute('id')).not.toBe(oldSvgId)
  await expectDiagrams(page, 5)

  await page.locator('a[href="/openlicensd/oidc-sso.html"]').first().click()
  await expect(page).toHaveURL(/\/oidc-sso\.html$/)
  await expectDiagrams(page, 1)
  await expect(page.locator('div.language-text pre').first()).toBeVisible()

  await page.locator('a[href="/openlicensd/harbor-registry-credentials.html"]').first().click()
  await expect(page).toHaveURL(/\/harbor-registry-credentials\.html$/)
  await expectDiagrams(page, 1)
  await page.locator('.VPSwitchAppearance:visible').first().click()
  await expect(page.locator('html')).not.toHaveClass(/dark/)
  await expectDiagrams(page, 1)

  await page.locator('a[href="/openlicensd/api-reference.html"]').first().click()
  await expect(page.getByRole('heading', { name: 'OpenAPI Reference' })).toBeVisible()
  await expect(page.getByText('OpenLicensd API', { exact: true }).first()).toBeVisible()
  expect(errors).toEqual([])
})
