import { test, expect, STREAM } from './fixtures'
import { call, publishRaw, streamLastSeq } from './api'

const MAPPINGS = 'natscope.mappings.v1.MappingsService'
const SOURCES = 'natscope.proto.sources.v1.SourcesService'
const SOURCE_NAME = 'e2e-pin-upload'
const SUBJECT = 'e2e.proto.pinned'

const schema = (field: string) => `syntax = "proto3";\npackage pin;\nmessage Note { string ${field} = 1; }\n`

async function cleanup() {
  const mappings = await call<{ mappings?: Array<{ id: string; pattern: string }> }>(MAPPINGS, 'ListMappings', { pageSize: 500 })
  for (const m of mappings.mappings ?? []) {
    if (m.pattern === SUBJECT) await call(MAPPINGS, 'DeleteMapping', { id: m.id })
  }
  const sources = await call<{ sources?: Array<{ id: string; name: string }> }>(SOURCES, 'ListSources', { pageSize: 500 })
  for (const s of sources.sources ?? []) {
    if (s.name === SOURCE_NAME) await call(SOURCES, 'DeleteSource', { id: s.id })
  }
}

async function upload(sourceId: string, field: string) {
  await call(SOURCES, 'UploadSchema', { sourceId, files: { files: [{ path: 'note.proto', content: schema(field) }] } })
}

async function savedPin(): Promise<string | undefined> {
  const list = await call<{ mappings?: Array<{ pattern: string; pinnedFingerprint?: string }> }>(MAPPINGS, 'ListMappings', {
    pageSize: 500,
  })
  return (list.mappings ?? []).find((m) => m.pattern === SUBJECT)?.pinnedFingerprint
}

test.describe('Mapping schema pins', () => {
  test.beforeEach(cleanup)
  test.afterAll(cleanup)

  test('a pinned mapping keeps decoding with its revision after the source moves on', async ({ page, env }) => {
    const created = await call<{ source?: { id: string } }>(SOURCES, 'CreateSource', {
      name: SOURCE_NAME,
      sourceType: 'SOURCE_TYPE_UPLOAD',
    })
    const sourceId = created.source?.id ?? ''
    await upload(sourceId, 'old_text')
    await call(MAPPINGS, 'CreateMapping', { pattern: SUBJECT, messageType: 'pin.Note', sourceId })

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

    await upload(sourceId, 'new_text')
    await publishRaw(env.connectionId, SUBJECT, SUBJECT, '\n\u0006pinned')
    const seq = await streamLastSeq(env.connectionId, STREAM)
    await page.goto(`/streams/${STREAM}/messages`)
    await page.locator('[role="row"]').filter({ hasText: SUBJECT }).first().click()
    const viewer = page.getByRole('complementary')
    await expect(viewer.getByText(`Message #${seq}`)).toBeVisible()
    await expect(viewer.getByRole('button', { name: 'old_text', exact: true })).toBeVisible()
    await expect(viewer.getByRole('button', { name: 'new_text' })).toBeHidden()

    await page.goto('/settings/mappings')
    await row.getByRole('button', { name: `Edit mapping ${SUBJECT}` }).click()
    await picker.click()
    await page.getByRole('option', { name: 'Active schema (follows every refresh)' }).click()
    await page.getByRole('button', { name: 'Save changes' }).click()
    await expect(row.getByTestId('mapping-pin')).toBeHidden()
    await expect.poll(savedPin).toBeFalsy()
  })

  test('a local source offers no revision to pin', async ({ page, env }) => {
    await call(MAPPINGS, 'CreateMapping', { pattern: SUBJECT, messageType: 'e2e.E2EMessage', sourceId: env.protoSourceId })
    await page.goto('/settings/mappings')
    await page.locator('tr').filter({ hasText: SUBJECT }).getByRole('button', { name: `Edit mapping ${SUBJECT}` }).click()
    await expect(page.locator('#mapping-schema-version')).toBeDisabled()
    await expect(page.getByText('A local directory keeps only its latest compile')).toBeVisible()
  })
})
