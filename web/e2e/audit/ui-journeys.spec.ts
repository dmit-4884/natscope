import { test, expect } from './fixtures-audit'
import * as A from './api-audit'
import { ns } from './fixtures-audit'

/**
 * Slice 16 — UI shell, routing & core UX states. Uses the connected-app
 * fixture (SPA boots already-connected to `local`). Seeds a namespaced stream
 * so the messages tab has a real success + empty state to render.
 */
const N = ns('ui')

test.describe('routing & shell', () => {
  test('CHK /streams index shows the empty "select a stream" prompt', async ({ page, env: _env }) => {
    await page.goto('/streams')
    await expect(page.getByText('Select a stream from the sidebar')).toBeVisible()
  })

  test('CHK /settings redirects to the connections section', async ({ page, env: _env }) => {
    await page.goto('/settings')
    await page.waitForURL(/\/settings\/connections/)
    await expect(page).toHaveURL(/\/settings\/connections/)
  })

  test('CHK /settings/preferences renders the preferences page', async ({ page, env: _env }) => {
    await page.goto('/settings/preferences')
    await expect(page.getByText('Defaults', { exact: false }).first()).toBeVisible()
  })

  test('CHK unknown route shows the 404 page', async ({ page, env: _env }) => {
    await page.goto('/this-route-does-not-exist-xyz')
    await expect(page.getByText('Page Not Found')).toBeVisible()
  })

  test('CHK legacy /storage/kv redirects to /kv', async ({ page, env: _env }) => {
    await page.goto('/storage/kv')
    await page.waitForURL(/\/kv$/)
    await expect(page).toHaveURL(/\/kv$/)
  })

  test('CHK legacy /management/streams redirects to /streams', async ({ page, env: _env }) => {
    await page.goto('/management/streams')
    await page.waitForURL(/\/streams$/)
    await expect(page).toHaveURL(/\/streams$/)
  })
})

test.describe('messages tab UI states', () => {
  test('CHK stream with seeded messages renders rows; empty stream renders empty state', async ({ page, env }) => {
    // Success state: seed a stream with messages.
    await A.deleteStreamIfExists(env.connectionId, N.stream)
    await A.createStream(env.connectionId, N.stream, { subjects: [N.wildcard], storage: 1 })
    for (let i = 0; i < 3; i++) {
      await A.publish(env.connectionId, N.subject(`m${i}`), N.wildcard, `{"i":${i}}`)
    }
    await page.goto(`/streams/${N.stream}/messages`)
    await expect(page.locator('#main-content').getByText(`${N.subjectRoot}.m0`, { exact: false }).first()).toBeVisible({ timeout: 15000 })

    // Empty state: a fresh stream with no messages.
    const empty = `${N.stream}_EMPTY`
    await A.deleteStreamIfExists(env.connectionId, empty)
    await A.createStream(env.connectionId, empty, { subjects: [`${N.subjectRoot}e.>`], storage: 1 })
    await page.goto(`/streams/${empty}/messages`)
    await expect(page.locator('#main-content').getByText(/no messages|empty|nothing/i).first()).toBeVisible({ timeout: 15000 })

    await A.deleteStreamIfExists(env.connectionId, N.stream)
    await A.deleteStreamIfExists(env.connectionId, empty)
  })
})
