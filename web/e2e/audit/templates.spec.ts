import { test, expect } from '@playwright/test'
import * as A from './api-audit'
import { ns } from './fixtures-audit'

/**
 * Slice 5 — Templates (API-level). Scoped to the AUDIT_TEMPLATES_ prefix;
 * never calls DeleteAllTemplates (would wipe unrelated templates).
 */
const N = ns('templates')
const tname = (s: string) => `${N.tplPrefix}${s}`

async function cleanup() {
  for (const t of (await A.listTemplates()).templates ?? []) {
    if (t.name.startsWith(N.tplPrefix)) await A.deleteTemplate(t.id).catch(() => {})
  }
}
test.beforeAll(cleanup)
test.afterAll(cleanup)

test.describe('templates: CRUD', () => {
  test('CHK create, get, list, update, delete', async () => {
    const created = await A.createTemplate({ name: tname('crud'), subject: 'audit.tpl.x', data: '{"a":1}', headers: { H: 'v' } })
    const id = (created.template as { id?: string }).id!
    expect((await A.listTemplates()).templates?.some((t) => t.id === id)).toBe(true)

    const got = (await A.call<{ template?: { name?: string; data?: string } }>(A.SVC.templates, 'GetTemplate', { id })).template
    expect(got?.name).toBe(tname('crud'))

    await A.call(A.SVC.templates, 'UpdateTemplate', { id, data: '{"a":2}' })
    const after = (await A.call<{ template?: { data?: string } }>(A.SVC.templates, 'GetTemplate', { id })).template
    expect(after?.data).toContain('"a":2')

    await A.deleteTemplate(id)
    expect(((await A.listTemplates()).templates ?? []).some((t) => t.id === id)).toBe(false)
  })
})

test.describe('templates: batch & validation', () => {
  test('CHK BatchCreateTemplates creates multiple', async () => {
    const res = await A.call<{ created?: number }>(A.SVC.templates, 'BatchCreateTemplates', {
      templates: [
        { name: tname('b1'), data: '{}' },
        { name: tname('b2'), data: '{}' },
      ],
    })
    expect(Number(res.created ?? '0')).toBe(2)
  })

  test('CHK [P2] empty template name rejected (min_len=1)', async () => {
    await expect(A.createTemplate({ name: '', data: '{}' })).rejects.toBeInstanceOf(A.ConnectError)
  })

  test('CHK [P2] GetTemplate with non-UUID id rejected (string.uuid)', async () => {
    await expect(A.call(A.SVC.templates, 'GetTemplate', { id: 'nope' })).rejects.toBeInstanceOf(A.ConnectError)
  })

  test('CHK [P2] negative page_size rejected (int32.gte=0)', async () => {
    await expect(A.listTemplates(-1)).rejects.toBeInstanceOf(A.ConnectError)
  })
})
