import { createPushConsumer, deleteConsumer, publishRaw } from './api'
import { test, expect, STREAM, PLAIN_SUBJECT } from './fixtures'

const CONSUMER = 'e2e-unread'

test.describe('consumers', () => {
  test.beforeEach(async ({ env }) => {
    await createPushConsumer(env.connectionId, STREAM, CONSUMER, PLAIN_SUBJECT, 'e2e.deliver.nobody')
    await publishRaw(env.connectionId, PLAIN_SUBJECT, PLAIN_SUBJECT, 'waiting for nobody')
  })

  test.afterEach(async ({ env }) => {
    await deleteConsumer(env.connectionId, STREAM, CONSUMER)
  })

  test('the overview explains a push consumer without a subscriber and opens it', async ({ page }) => {
    await page.goto('/request')
    await page.getByRole('link', { name: 'Consumers' }).click()
    await expect(page).toHaveURL(/\/consumers$/)
    await expect(page.getByTestId('consumers-summary')).toContainText('stuck')

    const row = page.getByRole('button', { name: new RegExp(`${CONSUMER}.*No subscriber`) })
    await expect(row).toBeVisible()

    await row.click()
    await expect(page).toHaveURL(new RegExp(`/streams/${STREAM}/consumers$`))
    await expect(page.getByRole('heading', { name: CONSUMER })).toBeVisible()
    await expect(page.getByTestId('consumer-problems')).toContainText('e2e.deliver.nobody')
  })

  test('the next message to deliver opens in the stream with its fate per consumer', async ({ page }) => {
    await page.goto(`/streams/${STREAM}/consumers?consumer=${CONSUMER}`)
    const next = page.getByTestId('consumer-position').getByRole('button', { name: /^Open message #\d+$/ }).last()
    await expect(next).toBeVisible()
    const label = (await next.getAttribute('aria-label')) ?? ''
    const sequence = label.replace('Open message #', '')

    await next.click()
    await expect(page).toHaveURL(new RegExp(`/streams/${STREAM}/messages$`))
    await expect(page.getByRole('heading', { name: `Message #${sequence}` })).toBeVisible()

    const fate = page.getByTestId('message-consumers')
    await expect(fate).toContainText('not delivered yet')
    await fate.getByRole('button', { name: /Consumers:/ }).click()
    await expect(fate.getByRole('listitem').filter({ hasText: CONSUMER })).toContainText('Not delivered yet')
  })

  test('the list exports as JSON', async ({ page }) => {
    await page.goto('/consumers')
    await expect(page.getByRole('button', { name: new RegExp(CONSUMER) })).toBeVisible()

    await page.getByRole('button', { name: 'Export consumers' }).click()
    const download = page.waitForEvent('download')
    await page.getByRole('menuitem', { name: 'Export as JSON' }).click()
    expect((await download).suggestedFilename()).toMatch(/^consumers-.*\.json$/)
  })
})
