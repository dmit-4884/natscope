import { test, expect, STREAM, PROTO_MESSAGE_TYPE } from './fixtures'
import { call, publishRaw, streamLastSeq } from './api'

const MAPPINGS = 'natscope.mappings.v1.MappingsService'
const SUBJECT = 'e2e.proto.framed'
const MESSAGE = '\n\u0005hello'
const GRPC_FRAMED = `\u0000\u0000\u0000\u0000${String.fromCharCode(MESSAGE.length)}${MESSAGE}`

async function deleteFramedMapping() {
  const list = await call<{ mappings?: Array<{ id: string; pattern: string }> }>(MAPPINGS, 'ListMappings', { pageSize: 500 })
  for (const m of list.mappings ?? []) {
    if (m.pattern === SUBJECT) await call(MAPPINGS, 'DeleteMapping', { id: m.id })
  }
}

test.describe('Mapping framing', () => {
  test.beforeEach(deleteFramedMapping)
  test.afterAll(deleteFramedMapping)

  test('strips a gRPC frame before decoding and is editable in the mapping form', async ({ page, env }) => {
    await call(MAPPINGS, 'CreateMapping', {
      pattern: SUBJECT,
      messageType: PROTO_MESSAGE_TYPE,
      sourceId: env.protoSourceId,
      framing: { kind: 'FRAMING_KIND_GRPC' },
    })
    await publishRaw(env.connectionId, SUBJECT, SUBJECT, GRPC_FRAMED)
    const seq = await streamLastSeq(env.connectionId, STREAM)

    await page.goto(`/streams/${STREAM}/messages`)
    await page.locator('[role="row"]').filter({ hasText: SUBJECT }).first().click()
    const viewer = page.getByRole('complementary')
    await expect(viewer.getByText(`Message #${seq}`)).toBeVisible()
    await expect(viewer.getByText('"hello"').first()).toBeVisible()

    await page.goto('/settings/mappings')
    const row = page.locator('tr').filter({ hasText: SUBJECT })
    await expect(row.getByTestId('mapping-framing')).toHaveText('gRPC frame')

    await row.getByRole('button', { name: `Edit mapping ${SUBJECT}` }).click()
    await expect(page.getByLabel('Framing')).toHaveValue('grpc')
    await page.getByLabel('Framing').selectOption('custom')
    await page.getByLabel('Prefix (hex)').fill('caf')
    await expect(page.getByRole('button', { name: 'Save changes' })).toBeDisabled()
    await page.getByLabel('Prefix (hex)').fill('cafe')
    await page.getByRole('button', { name: 'Save changes' }).click()
    await expect(row.getByTestId('mapping-framing')).toHaveText('custom')

    const list = await call<{ mappings?: Array<{ pattern: string; framing?: { kind?: string; prefix?: string } }> }>(
      MAPPINGS,
      'ListMappings',
      { pageSize: 500 },
    )
    const saved = (list.mappings ?? []).find((m) => m.pattern === SUBJECT)
    expect(saved?.framing).toEqual({ kind: 'FRAMING_KIND_CUSTOM', prefix: 'yv4=' })
  })
})
