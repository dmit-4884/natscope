import { test, expect, STREAM, PLAIN_SUBJECT } from './fixtures'
import { publishRaw, streamLastSeq } from './api'

const messagesUrl = `/streams/${STREAM}/messages`

async function selectNewest(page: import('@playwright/test').Page, subject: string, seq: number) {
  const row = page.locator('[role="row"]').filter({ hasText: subject }).first()
  await expect(row).toBeVisible()
  await row.click()
  await expect(page.getByRole('complementary').getByText(`Message #${seq}`)).toBeVisible()
}

test.describe('Edit & resend', () => {
  test('resend a wildcard-subject message → publish form prefilled, tab switched', async ({ page, env }) => {
    const marker = `resend-${Date.now()}`
    const subject = 'e2e.publish.eu.created' // matches pattern e2e.publish.*.*
    await publishRaw(env.connectionId, subject, 'e2e.publish.*.*', JSON.stringify({ order: marker }))
    const seq = await streamLastSeq(env.connectionId, STREAM)

    await page.goto(messagesUrl)
    await selectNewest(page, subject, seq)

    await page.getByTestId('resend-message').click()

    // Switched to the Publish tab.
    await expect(page).toHaveURL(/\/publish$/)

    // Wildcard slots recovered from the concrete subject.
    const slots = page.locator('#main-content input[placeholder="*"]')
    await expect(slots).toHaveCount(2)
    await expect(slots.nth(0)).toHaveValue('eu')
    await expect(slots.nth(1)).toHaveValue('created')

    // Payload carried over into the editor.
    await expect(page.locator('#main-content .cm-content')).toContainText(marker)
  })

  test('resend a literal-subject message → no wildcard slots, payload prefilled', async ({ page, env }) => {
    const marker = `resend-plain-${Date.now()}`
    await publishRaw(env.connectionId, PLAIN_SUBJECT, PLAIN_SUBJECT, JSON.stringify({ note: marker }))
    const seq = await streamLastSeq(env.connectionId, STREAM)

    await page.goto(messagesUrl)
    await selectNewest(page, PLAIN_SUBJECT, seq)

    await page.getByTestId('resend-message').click()
    await expect(page).toHaveURL(/\/publish$/)

    // A literal pattern has no wildcard slots.
    await expect(page.locator('#main-content input[placeholder="*"]')).toHaveCount(0)
    await expect(page.locator('#main-content .cm-content')).toContainText(marker)
  })
})
