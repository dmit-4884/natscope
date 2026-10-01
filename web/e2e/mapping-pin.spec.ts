import { test, expect, STREAM, PROTO_MESSAGE_TYPE } from './fixtures'
import { call, publishRaw, streamLastSeq } from './api'

const MAPPINGS = 'natscope.mappings.v1.MappingsService'
const SUBJECT = 'e2e.proto.pinned'

async function deletePinnedMapping() {
  const list = await call<{ mappings?: Array<{ id: string; pattern: string }> }>(MAPPINGS, 'ListMappings', { pageSize: 500 })
  for (const m of list.mappings ?? []) {
    if (m.pattern === SUBJECT) await call(MAPPINGS, 'DeleteMapping', { id: m.id })
  }
}

async function savedPin(): Promise<string | undefined> {
  const list = await call<{ mappings?: Array<{ pattern: string; pinnedFingerprint?: string }> }>(MAPPINGS, 'ListMappings', {
    pageSize: 500,
  })
  return (list.mappings ?? []).find((m) => m.pattern === SUBJECT)?.pinnedFingerprint
}

test.describe('Mapping schema pins', () => {
  test.beforeEach(deletePinnedMapping)
  test.afterAll(deletePinnedMapping)

  test('pins a mapping to a schema revision, decodes with it, and unpins', async ({ page, env }) => {
    await call(MAPPINGS, 'CreateMapping', { pattern: SUBJECT, messageType: PROTO_MESSAGE_TYPE, sourceId: env.protoSourceId })

    await page.goto('/settings/mappings')
    const row = page.locator('tr').filter({ hasText: SUBJECT })
    await expect(row.getByTestId('mapping-pin')).toBeHidden()

    await row.getByRole('button', { name: `Edit mapping ${SUBJECT}` }).click()
    const picker = page.locator('#mapping-schema-version')
    await expect(picker).toContainText('Active schema')
    await picker.click()
    await page.getByRole('option', { name: / · active$/ }).click()
    await page.getByRole('button', { name: 'Save changes' }).click()
    await expect(row.getByTestId('mapping-pin')).toBeVisible()
    await expect.poll(savedPin).toBeTruthy()

    await publishRaw(env.connectionId, SUBJECT, SUBJECT, '\n\u0006pinned')
    const seq = await streamLastSeq(env.connectionId, STREAM)
    await page.goto(`/streams/${STREAM}/messages`)
    await page.locator('[role="row"]').filter({ hasText: SUBJECT }).first().click()
    const viewer = page.getByRole('complementary')
    await expect(viewer.getByText(`Message #${seq}`)).toBeVisible()
    await expect(viewer.getByText('"pinned"').first()).toBeVisible()

    await page.goto('/settings/mappings')
    await row.getByRole('button', { name: `Edit mapping ${SUBJECT}` }).click()
    await picker.click()
    await page.getByRole('option', { name: 'Active schema (follows every refresh)' }).click()
    await page.getByRole('button', { name: 'Save changes' }).click()
    await expect(row.getByTestId('mapping-pin')).toBeHidden()
    await expect.poll(savedPin).toBeFalsy()
  })
})
