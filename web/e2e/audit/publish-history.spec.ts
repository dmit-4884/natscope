import { test, expect } from '@playwright/test'
import * as A from './api-audit'
import { ns } from './fixtures-audit'

/**
 * Slice 4 — Publish, dedup & publish history (API-level).
 * Namespaced stream AUDIT_PUBLISH with a small max_msg_size for oversize.
 */
const N = ns('publish')
const MAX = 1024
let cid: string

test.beforeAll(async () => {
  cid = await A.getConnectionId('local')
  await A.deleteStreamIfExists(cid, N.stream)
  await A.createStream(cid, N.stream, {
    subjects: [N.wildcard],
    storage: 1,
    maxMsgSize: MAX,
    duplicates: '120s',
  })
})
test.afterAll(async () => {
  await A.deleteStreamIfExists(cid, N.stream)
})

test.describe('publish: happy paths', () => {
  test('CHK publish raw string returns stream + sequence', async () => {
    const r = await A.publish(cid, N.subject('raw'), N.wildcard, 'plain-text')
    expect(r.error ?? '').toBe('')
    expect(Number(r.sequence ?? '0')).toBeGreaterThan(0)
    expect(r.stream).toBe(N.stream)
  })

  test('CHK publish JSON with custom headers', async () => {
    const r = await A.publish(cid, N.subject('json'), N.wildcard, '{"k":1}', { 'X-Audit': 'yes', 'X-Two': 'b' })
    expect(r.error ?? '').toBe('')
    const seq = Number(r.sequence ?? '0')
    const got = await A.getMessage(cid, N.stream, seq)
    expect(got.message?.headers?.['X-Audit']).toBe('yes')
    expect(A.msgText(got.message ?? {})).toContain('"k"')
  })

  test('CHK multibyte UTF-8 payload round-trips intact', async () => {
    const payload = '{"msg":"Привет 世界 🚀"}'
    const r = await A.publish(cid, N.subject('utf8'), N.wildcard, payload)
    const got = await A.getMessage(cid, N.stream, Number(r.sequence ?? '0'))
    expect(A.msgText(got.message ?? {})).toBe(payload)
  })
})

test.describe('publish: validation & limits', () => {
  test('CHK [P2] empty data rejected (buf.validate min_len=1)', async () => {
    await expect(A.publish(cid, N.subject('empty'), N.wildcard, '')).rejects.toBeInstanceOf(A.ConnectError)
  })

  test('CHK oversize payload (> max_msg_size) fails', async () => {
    const big = 'x'.repeat(MAX + 500)
    let failed: boolean
    try {
      const r = await A.publish(cid, N.subject('big'), N.wildcard, big)
      failed = !!(r.error && r.error.length > 0)
    } catch {
      failed = true
    }
    expect(failed, 'oversize publish should surface an error').toBe(true)
  })
})

test.describe('publish: dedup window', () => {
  test('CHK duplicate Nats-Msg-Id within window is flagged', async () => {
    const id = `audit-dup-${Date.now()}`
    const first = await A.publish(cid, N.subject('dup'), N.wildcard, '{"n":1}', { 'Nats-Msg-Id': id })
    expect(first.duplicate ?? false).toBe(false)
    const second = await A.call<{ duplicate?: boolean; sequence?: string }>(
      A.SVC.publish,
      'PublishMessage',
      { connectionId: cid, subject: N.subject('dup'), subjectPattern: N.wildcard, data: '{"n":2}', headers: { 'Nats-Msg-Id': id } },
    )
    expect(second.duplicate ?? false).toBe(true)
  })
})

test.describe('publish history', () => {
  test('CHK history records publishes for our stream', async () => {
    await A.publish(cid, N.subject('hist'), N.wildcard, '{"hist":true}')
    const res = await A.listPublishHistory({ pageSize: 200, stream: N.stream })
    const entries = res.entries ?? []
    expect(entries.length).toBeGreaterThan(0)
  })

  test('CHK [P2] negative page_size rejected (buf.validate int32.gte=0)', async () => {
    await expect(A.listPublishHistory({ pageSize: -1 })).rejects.toBeInstanceOf(A.ConnectError)
  })
})
