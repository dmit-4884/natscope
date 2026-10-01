import { mkdirSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { test, expect, PROTO_MESSAGE_TYPE } from './fixtures'
import { call } from './api'

const SOURCES = 'natscope.proto.sources.v1.SourcesService'
const SOURCE_NAME = 'e2e-clash'

async function deleteSource(): Promise<void> {
  const list = await call<{ sources?: Array<{ id: string; name: string }> }>(SOURCES, 'ListSources', { pageSize: 500 })
  for (const s of list.sources ?? []) {
    if (s.name === SOURCE_NAME) await call(SOURCES, 'DeleteSource', { id: s.id })
  }
}

test.beforeAll(async () => {
  await deleteSource()
  const dir = join(tmpdir(), 'natscope-e2e-clash')
  mkdirSync(dir, { recursive: true })
  writeFileSync(join(dir, 'clash.proto'), 'syntax = "proto3";\npackage e2e;\nmessage E2EMessage { int64 id = 1; }\n')
  const created = await call<{ source?: { id: string } }>(SOURCES, 'CreateSource', {
    name: SOURCE_NAME,
    sourceType: 'SOURCE_TYPE_LOCAL',
    localPath: dir,
    watcherEnabled: false,
  })
  await call(SOURCES, 'RefreshSource', { sourceId: created.source?.id })
})

test.afterAll(deleteSource)

test('the proto settings page lists types two sources define differently', async ({ page, env: _env }) => {
  await page.goto('/settings/proto')
  const conflict = page.getByTestId('schema-conflict').filter({ hasText: PROTO_MESSAGE_TYPE })
  await expect(conflict).toBeVisible()
  await expect(conflict).toContainText('Clash')
  await expect(conflict).toContainText('define this type differently')
  await expect(conflict).toContainText('e2e-proto-local · e2e.proto')
  await expect(conflict).toContainText(`${SOURCE_NAME} · clash.proto`)
})
