import { test, expect } from '@playwright/test'
import * as A from './api-audit'
import { ns } from './fixtures-audit'
import { ensureAuditProtoSource, deleteAuditProtoSource, MESSAGE_TYPE } from './generators/protoSource'

/**
 * Slice 9 — Subject mappings (API-level). Mappings require a proto source_id,
 * so a compiled audit source is created first. Cleans up by pattern prefix.
 * NOTE: BatchSaveMappings replaces ALL mappings atomically (global-destructive)
 * and is deliberately NOT exercised here — see the audit report.
 */
const N = ns('mappings')
const SRC = 'AUDIT_MAP_SRC'
let sourceId: string

async function cleanup() {
  for (const m of (await A.listMappings()).mappings ?? []) {
    if (m.pattern.startsWith(N.mappingPrefix)) await A.deleteMapping(m.id).catch(() => {})
  }
}

test.beforeAll(async () => {
  sourceId = await ensureAuditProtoSource(SRC)
  await cleanup()
})
test.afterAll(async () => {
  await cleanup()
  await deleteAuditProtoSource(SRC)
})

test.describe('mappings: CRUD', () => {
  test('CHK create, list, delete a mapping (MappingsService has no UpdateMapping RPC)', async () => {
    const created = await A.createMapping({ pattern: `${N.mappingPrefix}a.>`, messageType: MESSAGE_TYPE, sourceId })
    const id = created.mapping!.id
    expect(((await A.listMappings()).mappings ?? []).some((m) => m.id === id)).toBe(true)
    await A.deleteMapping(id)
    expect(((await A.listMappings()).mappings ?? []).some((m) => m.id === id)).toBe(false)
  })
})

test.describe('mappings: health', () => {
  test('CHK a mapping bound to a compiled source is healthy (ok)', async () => {
    const created = await A.createMapping({ pattern: `${N.mappingPrefix}health.>`, messageType: MESSAGE_TYPE, sourceId })
    const id = created.mapping!.id
    const res = await A.call<{ items?: Array<{ id?: string; health?: string }> }>(
      A.SVC.mappings,
      'BatchCheckMappingHealth',
      { ids: [id] },
    )
    const item = (res.items ?? []).find((i) => i.id === id)
    expect(item?.health).toBe('ok')
    await A.deleteMapping(id)
  })

  test('CHK a mapping to a missing type reports unhealthy', async () => {
    const created = await A.createMapping({ pattern: `${N.mappingPrefix}bad.>`, messageType: 'audit.codec.DoesNotExist', sourceId })
    const id = created.mapping!.id
    const res = await A.call<{ items?: Array<{ id?: string; health?: string }> }>(
      A.SVC.mappings,
      'BatchCheckMappingHealth',
      { ids: [id] },
    )
    const item = (res.items ?? []).find((i) => i.id === id)
    expect(item?.health).not.toBe('ok')
    await A.deleteMapping(id)
  })
})

test.describe('mappings: validation', () => {
  test('CHK [P2] empty pattern rejected (min_len=1)', async () => {
    await expect(A.createMapping({ pattern: '', messageType: MESSAGE_TYPE, sourceId })).rejects.toBeInstanceOf(A.ConnectError)
  })
  test('CHK [P2] missing source_id rejected (required)', async () => {
    await expect(A.createMapping({ pattern: `${N.mappingPrefix}nosrc.>`, messageType: MESSAGE_TYPE })).rejects.toBeInstanceOf(A.ConnectError)
  })
  test('CHK [P2] delete with non-UUID id rejected (string.uuid)', async () => {
    await expect(A.deleteMapping('not-a-uuid')).rejects.toBeInstanceOf(A.ConnectError)
  })
})
