import type { Page } from '@playwright/test'
import { test, expect, STREAM, PROTO_PATTERN } from './fixtures'
import { publishRaw, streamLastSeq } from './api'

const messagesUrl = `/streams/${STREAM}/messages`
const KNOWN = '\n\u0005newer'
const WITH_EXTRA_FIELD = `${KNOWN}x\u0003`
const BROKEN_TAIL = `${KNOWN}\u001a\u0009x`

async function openMessage(page: Page, subject: string, seq: number) {
  await page.goto(messagesUrl)
  const row = page.locator('[role="row"]').filter({ hasText: subject }).first()
  await expect(row).toBeVisible()
  await row.click()
  await expect(page.getByRole('complementary').getByText(`Message #${seq}`)).toBeVisible()
}

test.describe('Decode insights', () => {
  test('flags fields the schema does not declare and shows them on the wire', async ({ page, env }) => {
    const subject = 'e2e.proto.extra'
    await publishRaw(env.connectionId, subject, PROTO_PATTERN, WITH_EXTRA_FIELD)
    const seq = await streamLastSeq(env.connectionId, STREAM)

    await openMessage(page, subject, seq)
    const viewer = page.getByRole('complementary')
    await expect(viewer.getByTestId('decode-unknown')).toContainText('1 field not in the schema')
    await expect(viewer.getByText('"newer"').first()).toBeVisible()

    await viewer.getByRole('tab', { name: 'Wire' }).click()
    const fields = viewer.getByTestId('wire-field')
    await expect(fields).toHaveCount(2)
    await expect(fields.nth(0)).toContainText('"newer"')
    await expect(fields.nth(1)).toContainText('#15')
    await expect(fields.nth(1)).toContainText('varint')
  })

  test('decodes the valid prefix of a broken payload', async ({ page, env }) => {
    const subject = 'e2e.proto.broken'
    await publishRaw(env.connectionId, subject, PROTO_PATTERN, BROKEN_TAIL)
    const seq = await streamLastSeq(env.connectionId, STREAM)

    await openMessage(page, subject, seq)
    const viewer = page.getByRole('complementary')
    await expect(viewer.getByTestId('decode-partial')).toContainText('Decoded the first 7 B of 10 B')
    await expect(viewer.getByText('"newer"').first()).toBeVisible()

    await viewer.getByRole('tab', { name: 'Wire' }).click()
    await expect(viewer.getByTestId('wire-view')).toContainText('Stopped after 7 B of 10 B')
  })
})
