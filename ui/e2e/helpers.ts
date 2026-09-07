import { expect, type Locator, type Page } from '@playwright/test'

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
