import type { Page } from '@playwright/test'
import { publishRaw } from './api'
import { test, expect, STREAM, PLAIN_SUBJECT } from './fixtures'

async function openSearch(page: Page) {
  await page.goto(`/streams/${STREAM}/messages`)
  await page.getByRole('button', { name: 'Filters' }).click()
}

const rows = (page: Page) => page.getByRole('grid').getByRole('row')

test.describe('message search', () => {
  test('finds a message anywhere in the stream by its text, past the first page', async ({ page, env }) => {
    const token = `needle-${Date.now()}`
    await publishRaw(env.connectionId, PLAIN_SUBJECT, PLAIN_SUBJECT, `{"token":"${token}"}`)
    for (let i = 0; i < 60; i++) {
      await publishRaw(env.connectionId, PLAIN_SUBJECT, PLAIN_SUBJECT, `{"filler":${i}}`)
    }

    await openSearch(page)
    await page.getByLabel('Payload Search').fill(token.toUpperCase())
    await page.getByRole('button', { name: 'Apply' }).click()

    const status = page.getByTestId('search-status')
    await expect(status).toHaveAttribute('data-state', 'done')
    await expect(status).toContainText('· 1 found')
    await expect(rows(page)).toHaveCount(1)
    await expect(page.getByText('Content:')).toBeVisible()
  })

  test('a regular expression narrows the search', async ({ page, env }) => {
    const run = Date.now()
    for (let i = 1; i <= 3; i++) {
      await publishRaw(env.connectionId, PLAIN_SUBJECT, PLAIN_SUBJECT, `{"order":"ord-${run}-${i}"}`)
    }

    await openSearch(page)
    await page.getByLabel('Payload Search').fill(`ord-${run}-[12]"`)
    await page.getByRole('switch', { name: 'Regular expression' }).check({ force: true })
    await page.getByRole('button', { name: 'Apply' }).click()

    await expect(page.getByTestId('search-status')).toHaveAttribute('data-state', 'done')
    await expect(rows(page)).toHaveCount(2)
    await expect(page.getByText('Regex:')).toBeVisible()
  })

  test('a malformed regular expression explains itself', async ({ page, env: _env }) => {
    await openSearch(page)
    await page.getByLabel('Payload Search').fill('ord-(')
    await page.getByRole('switch', { name: 'Regular expression' }).check({ force: true })
    await page.getByRole('button', { name: 'Apply' }).click()

    const status = page.getByTestId('search-status')
    await expect(status).toHaveAttribute('data-state', 'error')
    await expect(status).toContainText('invalid regular expression')
  })

  test('removing the search filter returns to the normal list', async ({ page, env: _env }) => {
    await openSearch(page)
    await page.getByLabel('Payload Search').fill('anything-at-all')
    await page.getByRole('button', { name: 'Apply' }).click()
    await expect(page.getByTestId('search-status')).toBeVisible()

    await page.getByRole('button', { name: 'Remove Content filter' }).click()
    await expect(page.getByTestId('search-status')).toHaveCount(0)
    await expect(rows(page).first()).toBeVisible()
  })
})
