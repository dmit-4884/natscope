import type { Page } from '@playwright/test'
import { test, expect, STREAM } from './fixtures'

function streamsNav(page: Page) {
  return page.getByRole('navigation', { name: 'Streams navigation' })
}

test.describe('sidebar stream list', () => {
  test('filters streams by name', async ({ page, env: _env }) => {
    await page.goto('/streams')
    const nav = streamsNav(page)

    await nav.getByRole('textbox', { name: 'Filter streams' }).fill(STREAM.toLowerCase())

    await expect(nav.getByRole('link', { name: STREAM, exact: true })).toBeVisible()
    await nav.getByRole('textbox', { name: 'Filter streams' }).fill('no-such-stream-anywhere')
    await expect(nav.getByText('No streams match “no-such-stream-anywhere”')).toBeVisible()
  })

  test('a pinned stream stays on top across reloads', async ({ page, env: _env }) => {
    await page.goto('/streams')
    const nav = streamsNav(page)
    const pinned = nav.getByRole('list', { name: 'Pinned streams' })

    await nav.getByRole('textbox', { name: 'Filter streams' }).fill(STREAM)
    await nav.getByRole('button', { name: `Pin ${STREAM}` }).click()
    await expect(pinned.getByRole('link', { name: STREAM, exact: true })).toBeVisible()

    await page.reload()
    await expect(pinned.getByRole('link', { name: STREAM, exact: true })).toBeVisible()

    await nav.getByRole('button', { name: `Unpin ${STREAM}` }).click()
    await expect(pinned.getByRole('link', { name: STREAM, exact: true })).toHaveCount(0)
  })
})
