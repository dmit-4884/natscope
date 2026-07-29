import { test, expect } from '@playwright/test'
import * as A from './api-audit'
import { ns } from './fixtures-audit'

/**
 * Slice 13 — Workspace export/import (API-level). Non-destructive: the risky
 * paths use ValidateWorkspace (dry-run, never mutates); the real import path is
 * exercised with a throwaway mapping under MERGE (upsert-by-name, no deletes).
 */
const N = ns('workspace')
void N

test.describe('workspace: export & sections', () => {
  test('CHK ListSections returns self-describing sections with counts', async () => {
    const r = await A.listSections()
    const keys = (r.sections as Array<{ key?: string }>).map((s) => s.key)
    expect(keys).toContain('connections')
    expect(keys).toContain('mappings')
  })

  test('CHK ExportWorkspace returns JSON payload; secrets are excluded', async () => {
    const r = (await A.exportWorkspace({})) as { payload?: string }
    expect(r.payload, 'export payload present').toBeTruthy()
    const json = A.unb64(r.payload!).toString('utf8')
    // Must be parseable JSON and must not leak any secret value.
    const parsed = JSON.parse(json)
    expect(parsed).toBeTruthy()
    expect(json).not.toContain('super-secret')
  })

  test('CHK ExportWorkspace with explicit section_keys', async () => {
    const r = (await A.exportWorkspace({ sectionKeys: ['mappings'] })) as { payload?: string }
    expect(r.payload).toBeTruthy()
    JSON.parse(A.unb64(r.payload!).toString('utf8'))
  })
})

test.describe('workspace: validate (dry-run, non-mutating)', () => {
  test('CHK ValidateWorkspace MERGE reports without mutating', async () => {
    const exp = (await A.exportWorkspace({ sectionKeys: ['mappings'] })) as { payload?: string }
    const before = ((await A.listMappings()).mappings ?? []).length
    const rep = (await A.validateWorkspace({ payload: exp.payload, strategy: 'STRATEGY_MERGE' })) as { reports?: Array<Record<string, unknown>> }
    expect(Array.isArray(rep.reports)).toBe(true)
    const after = ((await A.listMappings()).mappings ?? []).length
    expect(after, 'validate must not change data').toBe(before)
  })

  test('CHK [P2] ValidateWorkspace REPLACE surfaces a deleted count without deleting', async () => {
    // REPLACE clears the section first; the dry-run surfaces `deleted` so the UI
    // can warn before a destructive import. Nothing is actually deleted by validate.
    const exp = (await A.exportWorkspace({ sectionKeys: ['connections'] })) as { payload?: string }
    const before = (await A.listConnections()).length
    const rep = (await A.validateWorkspace({ payload: exp.payload, strategy: 'STRATEGY_REPLACE' })) as { reports?: Array<{ key?: string; deleted?: number }> }
    const connReport = (rep.reports ?? []).find((r) => r.key === 'connections')
    expect(connReport).toBeTruthy()
    // Replacing the connections section would delete the existing connection(s).
    expect(Number(connReport?.deleted ?? 0)).toBeGreaterThanOrEqual(1)
    expect((await A.listConnections()).length, 'validate REPLACE must not actually delete').toBe(before)
  })

  test('CHK corrupted payload is rejected/reported, not silently accepted', async () => {
    let handled: boolean
    try {
      const rep = (await A.validateWorkspace({ payload: A.b64('this is not json {'), strategy: 'STRATEGY_MERGE' })) as { reports?: unknown[] }
      // If it does not throw, it must report an unknown/empty result, not pretend success.
      handled = !rep.reports || rep.reports.length === 0
    } catch {
      handled = true
    }
    expect(handled).toBe(true)
  })

  test('CHK [P2] empty payload rejected (buf.validate required=true)', async () => {
    await expect(A.validateWorkspace({ payload: '', strategy: 'STRATEGY_MERGE' })).rejects.toBeInstanceOf(A.ConnectError)
  })
})

test.describe('workspace: import (real path, MERGE, non-destructive)', () => {
  test('CHK ImportWorkspace MERGE of a full self-export is idempotent and preserves data', async () => {
    // Re-importing the exact current export under MERGE (upsert-by-name) must
    // not delete anything — the `local` connection and mapping count survive.
    const connBefore = (await A.listConnections()).length
    const mapBefore = ((await A.listMappings()).mappings ?? []).length
    const exp = (await A.exportWorkspace({})) as { payload?: string }
    const res = (await A.importWorkspace({ payload: exp.payload, strategy: 'STRATEGY_MERGE' })) as { results?: Array<{ key?: string; deleted?: number }> }
    expect(Array.isArray(res.results)).toBe(true)
    // No section should report deletions under MERGE.
    expect((res.results ?? []).every((r) => Number(r.deleted ?? 0) === 0)).toBe(true)
    expect((await A.listConnections()).some((c) => c.name === 'local')).toBe(true)
    expect((await A.listConnections()).length).toBe(connBefore)
    expect(((await A.listMappings()).mappings ?? []).length).toBe(mapBefore)
  })
})
