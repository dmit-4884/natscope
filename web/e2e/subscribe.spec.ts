import type { Page } from '@playwright/test'
import { publishCore, requestReply } from './api'
import { test, expect } from './fixtures'

async function startSubscription(page: Page, subject: string) {
  await page.goto('/subscribe')
  await expect(page.getByRole('heading', { name: 'Subscribe', exact: true })).toBeVisible()
  await page.getByLabel('Subjects').fill(subject)
  await page.getByLabel('Subjects').press('Enter')
  await page.getByRole('button', { name: 'Start' }).click()
  await expect(page.getByTestId('subscribe-status')).toHaveText('Live')
}

function feed(page: Page) {
  return page.getByRole('grid', { name: 'Received messages' })
}

test.describe('subscribe', () => {
  test('core messages on a wildcard subject show up as they are published', async ({ page, env }) => {
    const base = `e2e.sub.${Date.now()}`
    await startSubscription(page, `${base}.>`)

    await expect(async () => {
      await publishCore(env.connectionId, `${base}.orders`, '{"n":1}')
      await expect(feed(page).getByText(`${base}.orders`).first()).toBeVisible({ timeout: 1_000 })
    }).toPass({ timeout: 10_000 })

    await feed(page).getByText(`${base}.orders`).first().click()
    await expect(page.getByRole('complementary', { name: 'Details panel' })).toContainText('Object (1 key)')
  })

  test('stop keeps the received messages and start listens again', async ({ page, env }) => {
    const base = `e2e.sub.${Date.now()}`
    await startSubscription(page, `${base}.*`)
    await expect(async () => {
      await publishCore(env.connectionId, `${base}.a`, 'first')
      await expect(feed(page).getByText(`${base}.a`).first()).toBeVisible({ timeout: 1_000 })
    }).toPass({ timeout: 10_000 })

    await page.getByRole('button', { name: 'Stop' }).click()
    await expect(page.getByTestId('subscribe-stopped')).toBeVisible()
    await expect(feed(page).getByText(`${base}.a`).first()).toBeVisible()

    await page.getByRole('button', { name: 'Start' }).click()
    await expect(page.getByTestId('subscribe-status')).toHaveText('Live')
    await expect(feed(page).getByText(`${base}.a`).first()).toBeVisible()
    await expect(async () => {
      await publishCore(env.connectionId, `${base}.b`, 'second')
      await expect(feed(page).getByText(`${base}.b`).first()).toBeVisible({ timeout: 1_000 })
    }).toPass({ timeout: 10_000 })
    await expect(feed(page).getByText(`${base}.a`).first()).toBeVisible()
  })

  test('the subscription keeps listening while another page is open', async ({ page, env }) => {
    const base = `e2e.sub.${Date.now()}`
    await startSubscription(page, `${base}.>`)
    await expect(async () => {
      await publishCore(env.connectionId, `${base}.before`, '1')
      await expect(feed(page).getByText(`${base}.before`).first()).toBeVisible({ timeout: 1_000 })
    }).toPass({ timeout: 10_000 })

    await page.getByRole('link', { name: 'Request / Reply' }).click()
    await expect(page.getByRole('heading', { name: 'Request / Reply' })).toBeVisible()
    await expect(page.getByTestId('subscribe-listening')).toBeVisible()
    await publishCore(env.connectionId, `${base}.away`, '2')

    await page.getByRole('link', { name: /^Subscribe/ }).click()
    await expect(page.getByTestId('subscribe-status')).toHaveText('Live')
    await expect(feed(page).getByText(`${base}.before`).first()).toBeVisible()
    await expect(feed(page).getByText(`${base}.away`).first()).toBeVisible()
  })

  test('a stopped feed stays after visiting another page', async ({ page, env }) => {
    const base = `e2e.sub.${Date.now()}`
    await startSubscription(page, `${base}.>`)
    await expect(async () => {
      await publishCore(env.connectionId, `${base}.kept`, '1')
      await expect(feed(page).getByText(`${base}.kept`).first()).toBeVisible({ timeout: 1_000 })
    }).toPass({ timeout: 10_000 })
    await page.getByRole('button', { name: 'Stop' }).click()

    await page.getByRole('link', { name: 'Services' }).click()
    await expect(page.getByTestId('subscribe-listening')).toBeHidden()
    await page.getByRole('link', { name: /^Subscribe/ }).click()

    await expect(page.getByTestId('subscribe-stopped')).toBeVisible()
    await expect(feed(page).getByText(`${base}.kept`).first()).toBeVisible()
  })

  test('a request seen in the feed can be answered', async ({ page, env }) => {
    const base = `e2e.sub.${Date.now()}`
    await startSubscription(page, `${base}.>`)
    await expect(async () => {
      await publishCore(env.connectionId, `${base}.probe`, '')
      await expect(feed(page).getByText(`${base}.probe`).first()).toBeVisible({ timeout: 1_000 })
    }).toPass({ timeout: 10_000 })

    const reply = requestReply(env.connectionId, `${base}.ask`, '{"q":"ping"}', 20)
    await feed(page).getByText(`${base}.ask`).first().click()
    await page.getByTestId('reply-message').click()

    const dialog = page.getByRole('dialog', { name: 'Reply to request' })
    await dialog.locator('.cm-content').fill('pong')
    await dialog.getByRole('button', { name: 'Send reply' }).click()

    await expect(dialog).toBeHidden()
    expect(await reply).toBe('pong')
  })

  test('an invalid subject is rejected inline', async ({ page, env: _env }) => {
    await page.goto('/subscribe')
    await page.getByLabel('Subjects').fill('orders..x')
    await page.getByLabel('Subjects').press('Enter')

    await expect(page.getByTestId('subject-error')).toBeVisible()
    await expect(page.getByTestId('subject-chip')).toHaveCount(0)
  })
})
