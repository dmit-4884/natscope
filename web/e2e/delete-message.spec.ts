import { test, expect, STREAM, PLAIN_SUBJECT } from './fixtures'
import {
  publishRaw,
  streamLastSeq,
  messageExists,
  resetSettings,
  updateBehavior,
  snapshotSettings,
  restoreSettings,
} from './api'

const messagesUrl = `/streams/${STREAM}/messages`

// Click the newest (top) plain-subject row in the history list and wait for the
// viewer to show the expected sequence.
async function selectNewestPlain(page: import('@playwright/test').Page, seq: number) {
  const row = page.locator('[role="row"]').filter({ hasText: PLAIN_SUBJECT }).first()
  await expect(row).toBeVisible()
  await row.click()
  await expect(page.getByRole('complementary').getByText(`Message #${seq}`)).toBeVisible()
}

test.describe('Delete message', () => {
  let _settingsSnapshot: Awaited<ReturnType<typeof snapshotSettings>>

  test.beforeAll(async () => {
    _settingsSnapshot = await snapshotSettings()
    // Default state: confirmation ON.
    await resetSettings()
  })
  test.afterAll(async () => {
    await restoreSettings(_settingsSnapshot)
  })

  test('publish → confirm dialog → delete → gone from stream', async ({ page, env }) => {
    const marker = `del-${Date.now()}`
    await publishRaw(env.connectionId, PLAIN_SUBJECT, PLAIN_SUBJECT, JSON.stringify({ marker, note: 'delete-me' }))
    const seq = await streamLastSeq(env.connectionId, STREAM)
    expect(await messageExists(env.connectionId, STREAM, seq)).toBe(true)

    await page.goto(messagesUrl)
    await selectNewestPlain(page, seq)

    // Open the delete dialog and confirm a normal delete.
    await page.getByTestId('delete-message').click()
    await expect(page.getByTestId('delete-mode')).toBeVisible()
    await page.getByTestId('confirm-delete-message-btn').click()

    await expect(page.locator('[data-sonner-toast]').getByText(`Message #${seq} deleted`)).toBeVisible()
    // Selection clears back to the empty viewer state.
    await expect(page.getByText('Select a message to view details')).toBeVisible()

    // Backend confirms the message is actually gone.
    expect(await messageExists(env.connectionId, STREAM, seq)).toBe(false)
  })

  test('confirmation disabled → delete happens without a dialog', async ({ page, env }) => {
    // Opt out of the message-delete confirmation.
    await updateBehavior({ confirmDeleteMessage: false })

    const marker = `del-skip-${Date.now()}`
    await publishRaw(env.connectionId, PLAIN_SUBJECT, PLAIN_SUBJECT, JSON.stringify({ marker, note: 'delete-me' }))
    const seq = await streamLastSeq(env.connectionId, STREAM)

    await page.goto(messagesUrl)
    await selectNewestPlain(page, seq)

    // No dialog: clicking delete removes the message straight away.
    await page.getByTestId('delete-message').click()
    await expect(page.getByTestId('delete-mode')).toHaveCount(0)
    await expect(page.locator('[data-sonner-toast]').getByText(`Message #${seq} deleted`)).toBeVisible()
    expect(await messageExists(env.connectionId, STREAM, seq)).toBe(false)
  })
})
