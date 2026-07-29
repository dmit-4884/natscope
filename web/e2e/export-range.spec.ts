import { readFileSync } from 'node:fs'
import { test, expect, STREAM, PLAIN_SUBJECT } from './fixtures'
import { publishRaw, streamMessageCount, resetSettings, snapshotSettings, restoreSettings } from './api'

const messagesUrl = `/streams/${STREAM}/messages`

async function openExportRange(page: import('@playwright/test').Page) {
  await page.goto(messagesUrl)
  await page.getByRole('button', { name: 'Export' }).click()
  await page.getByTestId('export-scope-range').check()
}

async function downloadExport(page: import('@playwright/test').Page): Promise<string> {
  const [download] = await Promise.all([
    page.waitForEvent('download'),
    page.getByTestId('export-confirm').click(),
  ])
  const path = await download.path()
  return readFileSync(path, 'utf-8')
}

test.describe('Export full range', () => {
  let _settingsSnapshot: Awaited<ReturnType<typeof snapshotSettings>>

  test.beforeAll(async () => {
    _settingsSnapshot = await snapshotSettings()
    await resetSettings() // default export format = json
  })
  test.afterAll(async () => {
    await restoreSettings(_settingsSnapshot)
  })

  test('full-range export downloads every present message from the server', async ({ page, env }) => {
    // Seed a few messages so the stream is non-empty and stable for this test.
    for (let i = 0; i < 3; i++) {
      await publishRaw(env.connectionId, PLAIN_SUBJECT, PLAIN_SUBJECT, JSON.stringify({ range: i }))
    }
    const count = await streamMessageCount(env.connectionId, STREAM)

    await openExportRange(page)
    const content = await downloadExport(page)

    const arr = JSON.parse(content) as unknown[]
    expect(Array.isArray(arr)).toBe(true)
    // The high default limit means the whole stream is exported.
    expect(arr.length).toBe(count)
    await expect(page.locator('[data-sonner-toast]').getByText(/Exported .* messages/)).toBeVisible()
  })

  test('range export respects the limit and reports truncation', async ({ page, env }) => {
    for (let i = 0; i < 4; i++) {
      await publishRaw(env.connectionId, PLAIN_SUBJECT, PLAIN_SUBJECT, JSON.stringify({ trunc: i }))
    }

    await openExportRange(page)
    await page.getByTestId('export-limit').fill('2')
    const content = await downloadExport(page)

    const arr = JSON.parse(content) as unknown[]
    expect(arr.length).toBe(2)
    await expect(
      page.locator('[data-sonner-toast]').getByText(/Exported first 2 messages \(limit reached\)/),
    ).toBeVisible()
  })
})
