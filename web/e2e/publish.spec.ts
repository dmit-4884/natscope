import type { Page } from '@playwright/test'
import { test, expect, publishUrl, PATTERN, PLAIN_SUBJECT, STREAM } from './fixtures'
import { publishMessage, deleteTemplatesByPrefix } from './api'

const TPL_PREFIX = 'E2E-TPL'

async function openPublish(page: Page) {
  await page.goto(publishUrl)
  await expect(page.locator('#main-content').getByText('Subject Pattern', { exact: true })).toBeVisible()
}

async function selectPattern(page: Page, pattern: string) {
  const trigger = page.locator('#main-content button', { hasText: /Select Subject Pattern|e2e\./ }).first()
  await trigger.click()
  await page.getByRole('button', { name: pattern, exact: true }).click()
}

function editor(page: Page) {
  // CodeMirror 6 renders a contenteditable .cm-content instead of a textarea.
  return page.locator('#main-content .cm-content')
}

function history(page: Page) {
  return page.getByRole('complementary')
}

async function setPayload(page: Page, json: string) {
  await editor(page).fill(json)
}

test.describe('publish tab', () => {
  test('smoke: form renders, Validate button is gone in JSON mode', async ({ page, env: _env }) => {
    await openPublish(page)
    await selectPattern(page, PLAIN_SUBJECT)
    await expect(editor(page)).toBeVisible()
    await expect(page.getByTestId('templates-trigger')).toBeVisible()
    // JSON mode has no schema — the old always-disabled Validate button must not render.
    await expect(page.getByRole('button', { name: 'Validate' })).toHaveCount(0)
    await expect(page.getByTestId('validation-badge')).toHaveCount(0)
  })

  test('headers persist across tab switches and reloads', async ({ page, env: _env }) => {
    await openPublish(page)
    await selectPattern(page, PLAIN_SUBJECT)

    await page.getByTestId('add-header').click()
    await page.getByPlaceholder('Key').fill('X-E2E')
    await page.locator('input[placeholder="Value"]').fill('persisted')

    // Switch to Messages and back — headers must survive the unmount.
    await page.getByRole('link', { name: 'Messages', exact: true }).click()
    await page.getByRole('link', { name: 'Publish', exact: true }).click()
    await expect(page.getByPlaceholder('Key')).toHaveValue('X-E2E')

    // Full reload — headers live in the localStorage draft.
    await page.reload()
    await expect(page.getByPlaceholder('Key')).toHaveValue('X-E2E')
    await expect(page.locator('input[placeholder="Value"]')).toHaveValue('persisted')
  })

  test('Cmd/Ctrl+Enter publishes from a wildcard input', async ({ page, env: _env }) => {
    await openPublish(page)
    await selectPattern(page, PATTERN)

    const slots = page.locator('#main-content input[placeholder="*"]')
    await slots.nth(0).fill('hotkey')
    await slots.nth(1).fill('test')
    await setPayload(page, '{"via":"hotkey"}')

    await slots.nth(1).focus()
    await page.keyboard.press('ControlOrMeta+Enter')

    await expect(page.locator('[data-sonner-toast]').getByText(/Message published/)).toBeVisible()
  })

  test('auto encoding chip is visible and disappears when mode is pinned', async ({ page, env: _env }) => {
    await openPublish(page)
    await selectPattern(page, PLAIN_SUBJECT)

    await expect(page.getByTestId('encoding-auto-chip')).toBeVisible()
    await page.getByRole('button', { name: 'JSON', exact: true }).click()
    await expect(page.getByTestId('encoding-auto-chip')).toHaveCount(0)
  })

  test('byte counter shows size and warns above stream max_msg_size', async ({ page, env: _env }) => {
    await openPublish(page)
    await selectPattern(page, PLAIN_SUBJECT)

    await setPayload(page, '{"small":1}')
    await expect(page.getByTestId('payload-size')).toContainText('B / 1.0 KB')
    await expect(page.getByTestId('payload-oversize-warning')).toHaveCount(0)

    const big = JSON.stringify({ big: 'x'.repeat(1500) })
    await setPayload(page, big)
    await expect(page.getByTestId('payload-oversize-warning')).toBeVisible()
    await expect(page.getByTestId('payload-size')).toContainText('KB / 1.0 KB')
  })

  test('failed publish lands in history with the error text', async ({ page, env: _env }) => {
    await openPublish(page)
    await selectPattern(page, PLAIN_SUBJECT)

    const marker = `fail-${Date.now()}`
    await setPayload(page, JSON.stringify({ marker, pad: 'x'.repeat(1500) }))
    await page.getByRole('button', { name: 'Publish Message' }).click()

    await expect(page.locator('[data-sonner-toast]').getByText(/Error/)).toBeVisible()

    // Narrow the history list down to exactly this attempt (search runs over
    // payload JSON), then expand it — the error text must be shown.
    await page.getByPlaceholder('Search history...').fill(marker)
    const entry = history(page).getByText(PLAIN_SUBJECT).first()
    await expect(entry).toBeVisible()
    await entry.click()
    await expect(page.getByTestId('history-entry-error')).toBeVisible()
  })

  test('history: load into form restores subject, wildcards and payload', async ({ page, env: _env, context }) => {
    await context.grantPermissions(['clipboard-read', 'clipboard-write'])
    await openPublish(page)
    await selectPattern(page, PATTERN)

    const marker = `reload-${Date.now()}`
    const slots = page.locator('#main-content input[placeholder="*"]')
    await slots.nth(0).fill('lotX')
    await slots.nth(1).fill('userY')
    await setPayload(page, JSON.stringify({ marker }))
    await page.getByRole('button', { name: 'Publish Message' }).click()
    await expect(page.locator('[data-sonner-toast]').getByText(/Message published/)).toBeVisible()

    // Dirty the form, then restore from history.
    await slots.nth(0).fill('changed')
    await setPayload(page, '{}')

    await page.getByPlaceholder('Search history...').fill(marker)
    const entry = history(page).getByText('e2e.publish.lotX.userY').first()
    await entry.click()
    await page.getByTestId('history-load-into-form').click()

    await expect(editor(page)).toContainText(marker)
    await expect(slots.nth(0)).toHaveValue('lotX')
    await expect(slots.nth(1)).toHaveValue('userY')

    // Copy payload puts pretty-printed JSON into the clipboard.
    await page.getByTestId('history-copy-payload').click()
    const clipboard = await page.evaluate(() => navigator.clipboard.readText())
    expect(clipboard).toContain(marker)
  })

  test('templates: save, cross-subject load fills payload, undo restores draft', async ({ page, env: _env }) => {
    const tplName = `${TPL_PREFIX}-${Date.now()}`
    await openPublish(page)
    await selectPattern(page, PATTERN)

    const slots = page.locator('#main-content input[placeholder="*"]')
    await slots.nth(0).fill('tpl')
    await slots.nth(1).fill('slot')
    await setPayload(page, '{"from":"template"}')

    await page.getByTestId('save-template').click()
    await page.getByPlaceholder('e.g. orders.create — happy path').fill(tplName)
    await page.getByRole('button', { name: 'Save template' }).click()
    await expect(page.locator('[data-sonner-toast]').getByText(/saved/)).toBeVisible()

    // Move the form to a different subject + payload entirely.
    await selectPattern(page, PLAIN_SUBJECT)
    await setPayload(page, '{"other":"draft"}')

    // Load the template back: subject pattern AND payload must switch together
    // (regression: payload used to land in the previous pattern's draft).
    await page.getByTestId('templates-trigger').click()
    await page.getByPlaceholder('Search by name, subject or type…').fill(tplName)
    await page.getByText(tplName, { exact: true }).click()

    await expect(editor(page)).toContainText(/from.*template/)
    await expect(slots.nth(0)).toHaveValue('tpl')

    // Undo from the toast brings the previous draft back.
    await page.getByRole('button', { name: 'Undo' }).click()
    await expect(editor(page)).toContainText(/other.*draft/)

    await deleteTemplatesByPrefix(TPL_PREFIX)
  })

  test('headers editor suggests standard NATS headers and flags duplicates', async ({ page, env: _env }) => {
    await openPublish(page)
    await selectPattern(page, PLAIN_SUBJECT)

    await page.getByTestId('add-header').click()
    const firstKey = page.getByPlaceholder('Key').first()
    await firstKey.fill('Nats-Msg-Id')
    await expect(page.getByTestId('header-hint')).toContainText('de-duplication')
    // Known header suggests a helper value.
    await expect(page.locator('input[placeholder="{{uuid}}"]')).toBeVisible()

    await page.getByTestId('add-header').click()
    await page.getByPlaceholder('Key').nth(1).fill('Nats-Msg-Id')
    await expect(page.getByTestId('header-duplicate-warning').first()).toBeVisible()
  })

  test('prefill from last message fills the editor with real data', async ({ page, env }) => {
    const marker = `seeded-${Date.now()}`
    await publishMessage(env.connectionId, PLAIN_SUBJECT, PLAIN_SUBJECT, { marker })

    await openPublish(page)
    await selectPattern(page, PLAIN_SUBJECT)
    await setPayload(page, '{}')

    await page.getByTestId('prefill-from-last').click()
    await expect(page.locator('[data-sonner-toast]').getByText(/Loaded message #/)).toBeVisible()
    await expect(editor(page)).toContainText(marker)
  })

  test('publish history entry shows sequence for successful publishes', async ({ page, env: _env }) => {
    await openPublish(page)
    await selectPattern(page, PLAIN_SUBJECT)
    await setPayload(page, `{"seq_check":"${Date.now()}"}`)
    await page.getByRole('button', { name: 'Publish Message' }).click()
    await expect(page.locator('[data-sonner-toast]').getByText(/Message published/)).toBeVisible()

    const entry = history(page).getByText(PLAIN_SUBJECT).first()
    await entry.click()
    await expect(history(page).getByText(/seq \d+/)).toBeVisible()
  })
})

test.afterAll(async () => {
  await deleteTemplatesByPrefix(TPL_PREFIX).catch(() => {})
})

// Keep STREAM referenced for documentation purposes in test titles/reports.
void STREAM
