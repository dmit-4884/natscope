import type { Page } from '@playwright/test'
import { test, expect, STREAM, PATTERN, PROTO_MESSAGE_TYPE } from './fixtures'
import {
  deleteMappingsByPrefix,
  getSettings,
  mappingExists,
  publishRaw,
  resetSettings,
  restoreSettings,
  snapshotSettings,
  streamLastSeq,
  updateMessageSettings,
} from './api'

const messagesUrl = `/streams/${STREAM}/messages`
const PREFIX = 'e2e.publish.detect'

function e2eMessage(name: string): string {
  return `\n${String.fromCharCode(name.length)}${name}\u0010\u0003\u001a\u0001x`
}

async function openMessage(page: Page, subject: string, seq: number) {
  await page.goto(messagesUrl)
  const row = page.locator('[role="row"]').filter({ hasText: subject }).first()
  await expect(row).toBeVisible()
  await row.click()
  await expect(page.getByRole('complementary').getByText(`Message #${seq}`)).toBeVisible()
}

let snapshot: Awaited<ReturnType<typeof snapshotSettings>>
test.beforeAll(async () => {
  snapshot = await snapshotSettings()
  await resetSettings()
  await deleteMappingsByPrefix(PREFIX)
})
test.afterAll(async () => {
  await deleteMappingsByPrefix(PREFIX)
  await restoreSettings(snapshot)
})

test.describe.serial('Type detection', () => {
  test('decodes an unmapped binary payload and saves the guess as a mapping', async ({ page, env }) => {
    const subject = `${PREFIX}.1001`
    await publishRaw(env.connectionId, subject, PATTERN, e2eMessage('guessed'))
    const seq = await streamLastSeq(env.connectionId, STREAM)

    await openMessage(page, subject, seq)
    const viewer = page.getByRole('complementary')
    await expect(viewer.getByText(PROTO_MESSAGE_TYPE, { exact: true })).toBeVisible()
    await expect(viewer.getByTestId('decoded-auto')).toBeVisible()
    await expect(viewer.getByText('"guessed"').first()).toBeVisible()

    await viewer.getByTestId('resend-message').click()
    await expect(page.locator('[data-sonner-toast]').getByText(/Save the detected type as a mapping first/)).toBeVisible()

    await viewer.getByTestId('save-detected-mapping').click()
    await expect(page.locator('[data-sonner-toast]').getByText(`Mapped ${PREFIX}.* to ${PROTO_MESSAGE_TYPE}`)).toBeVisible()
    await expect(viewer.getByTestId('decoded-auto')).toBeHidden()
    await expect.poll(() => mappingExists(`${PREFIX}.*`)).toBe(true)
  })

  test('the settings toggle turns detection off', async ({ page, env }) => {
    void env
    await page.goto('/settings/preferences')
    const toggle = page.getByTestId('detect-types')
    await expect(toggle).toBeChecked()
    await toggle.uncheck({ force: true })
    await page.getByRole('button', { name: 'Save Changes' }).click()
    await expect(page.locator('[data-sonner-toast]').getByText(/Settings saved/)).toBeVisible()
    await expect.poll(async () => (await getSettings()).settings?.messages?.detectTypes).toBe(false)
  })

  test('Detect type ranks candidates for a raw payload', async ({ page, env }) => {
    await updateMessageSettings({ detectTypes: false })
    const subject = 'e2e.publish.manual.detect'
    await publishRaw(env.connectionId, subject, PATTERN, e2eMessage('picked'))
    const seq = await streamLastSeq(env.connectionId, STREAM)

    await openMessage(page, subject, seq)
    const viewer = page.getByRole('complementary')
    await expect(viewer.getByText('Not configured')).toBeVisible()
    await viewer.getByTestId('detect-type').click()

    const dialog = page.getByRole('dialog', { name: 'Detect message type' })
    const first = dialog.getByTestId('type-candidate').first()
    await expect(first).toContainText(PROTO_MESSAGE_TYPE)
    await expect(first).toContainText('% fit')
    await expect(dialog.getByTestId('detect-preview')).toContainText('"picked"')
    await expect(dialog.getByTestId('detect-pattern')).toHaveValue(subject)

    await dialog.getByTestId('detect-use').click()
    await expect(dialog).toBeHidden()
    await expect(viewer.getByText(PROTO_MESSAGE_TYPE, { exact: true })).toBeVisible()
    await expect(viewer.getByText('"picked"').first()).toBeVisible()
    await expect(viewer.getByTestId('decoded-auto')).toBeHidden()
    await expect(viewer.getByTestId('save-detected-mapping')).toBeVisible()
  })
})
