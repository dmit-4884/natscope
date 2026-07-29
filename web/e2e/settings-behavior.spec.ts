import { test, expect } from './fixtures'
import { resetSettings, snapshotSettings, restoreSettings } from './api'

const PREFERENCES_URL = '/settings/preferences'

// Behavior settings are a global singleton on the backend — snapshot before
// the suite runs and restore after, so a developer's live config survives.
let _settingsSnapshot: Awaited<ReturnType<typeof snapshotSettings>>
test.beforeAll(async () => {
  _settingsSnapshot = await snapshotSettings()
  await resetSettings()
})
test.afterAll(async () => {
  await restoreSettings(_settingsSnapshot)
})

test.describe('Behavior settings — confirm toggles', () => {
  test('toggling a confirmation off persists across reload, siblings untouched', async ({ page, env }) => {
    void env // env seeds localStorage so the app is reachable
    await page.goto(PREFERENCES_URL)

    const confirmMessage = page.getByTestId('confirm-delete-message')
    const confirmConsumer = page.getByTestId('confirm-delete-consumer')

    // Defaults: every confirmation ON.
    await expect(confirmMessage).toBeChecked()
    await expect(confirmConsumer).toBeChecked()

    // Turn the message confirmation OFF and save.
    await confirmMessage.uncheck({ force: true })
    await page.getByRole('button', { name: 'Save Changes' }).click()
    await expect(page.locator('[data-sonner-toast]').getByText(/Settings saved/)).toBeVisible()

    // Reload: the toggle stays off; the sibling stays on (partial update).
    await page.reload()
    await expect(page.getByTestId('confirm-delete-message')).not.toBeChecked()
    await expect(page.getByTestId('confirm-delete-consumer')).toBeChecked()
  })

  test('"Reset all confirmations" re-enables every prompt', async ({ page, env }) => {
    void env
    // Seed a disabled state directly so the test is independent of ordering.
    await resetSettings()
    await page.goto(PREFERENCES_URL)

    // Disable two confirmations and save.
    await page.getByTestId('confirm-delete-message').uncheck({ force: true })
    await page.getByTestId('confirm-delete-object').uncheck({ force: true })
    await page.getByRole('button', { name: 'Save Changes' }).click()
    await expect(page.locator('[data-sonner-toast]').getByText(/Settings saved/)).toBeVisible()

    // Reset all confirmations, then save.
    await page.getByTestId('reset-confirmations').click()
    await expect(page.getByTestId('confirm-delete-message')).toBeChecked()
    await expect(page.getByTestId('confirm-delete-object')).toBeChecked()
    await page.getByRole('button', { name: 'Save Changes' }).click()
    await expect(page.locator('[data-sonner-toast]').getByText(/Settings saved/)).toBeVisible()

    // Persisted: all confirmations back on after reload.
    await page.reload()
    await expect(page.getByTestId('confirm-delete-consumer')).toBeChecked()
    await expect(page.getByTestId('confirm-delete-message')).toBeChecked()
    await expect(page.getByTestId('confirm-delete-kv-key')).toBeChecked()
    await expect(page.getByTestId('confirm-delete-object')).toBeChecked()
    await expect(page.getByTestId('confirm-purge-kv-history')).toBeChecked()
  })
})
