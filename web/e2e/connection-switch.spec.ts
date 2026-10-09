import type { Page } from '@playwright/test'
import { deleteConnectionsByPrefix, ensureReadOnlyCopy } from './api'
import { test, expect, STREAM } from './fixtures'

const OTHER = 'e2e-switch'

async function switchTo(page: Page, name: RegExp) {
  await page.getByRole('banner').getByRole('button', { name: /nats:\/\// }).first().click()
  await page.getByRole('banner').getByRole('button', { name }).first().click()
}

test.describe('switching connections', () => {
  test.afterEach(async () => {
    await deleteConnectionsByPrefix(OTHER)
  })

  test('a connection visited before shows its sidebar lists at once', async ({ page, env }) => {
    await ensureReadOnlyCopy(env.connectionId, OTHER)
    await page.goto('/streams')
    const nav = page.getByRole('navigation', { name: 'Streams navigation' })
    await expect(nav.getByRole('link', { name: STREAM })).toBeVisible()

    await switchTo(page, new RegExp(`connection ${OTHER}`))
    await expect(page.getByRole('banner').getByText('PROD')).toBeVisible()
    await expect(nav.getByRole('link', { name: STREAM })).toBeVisible()

    await page.evaluate(() => {
      const w = window as unknown as { sawLoading: boolean }
      w.sawLoading = false
      new MutationObserver(() => {
        if (document.querySelector('nav [aria-busy="true"]')) w.sawLoading = true
      }).observe(document.body, { subtree: true, childList: true, attributes: true })
    })
    await switchTo(page, /connection local\b/)
    await expect(page.getByRole('banner').getByText('PROD')).toBeHidden()
    await expect(nav.getByRole('link', { name: STREAM })).toBeVisible()

    expect(await page.evaluate(() => (window as unknown as { sawLoading: boolean }).sawLoading)).toBe(false)
  })
})
