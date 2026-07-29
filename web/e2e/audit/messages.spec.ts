import { test, expect } from '@playwright/test'
import * as A from './api-audit'
import { ns } from './fixtures-audit'

/**
 * Slice 3 — Message browsing (API-level): pagination, filters, direction,
 * start_seq/start_time XOR (CEL), payload truncation, Get by seq, not-found.
 */
const N = ns('messages')
let cid: string

test.beforeAll(async () => {
  cid = await A.getConnectionId('local')
  await A.deleteStreamIfExists(cid, N.stream)
  await A.createStream(cid, N.stream, { subjects: [N.wildcard], storage: 1 })
  // Seed 6 messages across two leaf subjects.
  for (let i = 1; i <= 6; i++) {
    await A.publish(cid, N.subject(i % 2 === 0 ? `even.${i}` : `odd.${i}`), N.wildcard, `{"n":${i},"tag":"seed${i}"}`)
  }
})
test.afterAll(async () => {
  await A.deleteStreamIfExists(cid, N.stream)
})

test.describe('messages: list & pagination', () => {
  test('CHK forward list returns ascending sequences', async () => {
    const { messages } = await A.listMessages(cid, N.stream, { direction: 'DIRECTION_FORWARD', limit: '10' })
    expect(messages.length).toBe(6)
    const seqs = messages.map((m) => Number(m.sequence))
    expect(seqs).toEqual([...seqs].sort((a, b) => a - b))
  })

  test('CHK limit + cursor paging via nextSeq/hasMore', async () => {
    const page1 = await A.listMessages(cid, N.stream, { direction: 'DIRECTION_FORWARD', limit: '3' })
    expect(page1.messages.length).toBe(3)
    expect(page1.hasMore).toBe(true)
    const page2 = await A.listMessages(cid, N.stream, { direction: 'DIRECTION_FORWARD', limit: '3', startSeq: page1.nextSeq })
    expect(page2.messages.length).toBeGreaterThan(0)
    // No overlap between pages.
    const s1 = new Set(page1.messages.map((m) => m.sequence))
    expect(page2.messages.every((m) => !s1.has(m.sequence))).toBe(true)
  })

  test('CHK backward list returns descending sequences (newest first)', async () => {
    const { messages } = await A.listMessages(cid, N.stream, { direction: 'DIRECTION_BACKWARD', limit: '10' })
    const seqs = messages.map((m) => Number(m.sequence))
    expect(seqs).toEqual([...seqs].sort((a, b) => b - a))
  })
})

test.describe('messages: filters', () => {
  test('CHK subjectFilter narrows to matching subjects', async () => {
    const { messages } = await A.listMessages(cid, N.stream, { subjectFilter: `${N.subjectRoot}.even.>`, limit: '10' })
    expect(messages.length).toBeGreaterThan(0)
    expect(messages.every((m) => (m.subject ?? '').includes('.even.'))).toBe(true)
  })

  test('CHK contentFilter matches payload substring', async () => {
    const { messages } = await A.listMessages(cid, N.stream, { contentFilter: 'seed3', limit: '10' })
    expect(messages.length).toBe(1)
    expect(A.msgText(messages[0])).toContain('seed3')
  })
})

test.describe('messages: validation & truncation', () => {
  test('CHK [P2] start_seq and start_time together rejected (CEL xor)', async () => {
    await expect(
      A.listMessages(cid, N.stream, { startSeq: '1', startTime: new Date().toISOString() }),
    ).rejects.toBeInstanceOf(A.ConnectError)
  })

  test('CHK maxPayloadBytes truncates list payload but preserves dataSize; Get returns full', async () => {
    const big = `{"blob":"${'y'.repeat(2000)}"}`
    const pub = await A.publish(cid, N.subject('big'), N.wildcard, big)
    const seq = Number(pub.sequence)
    const { messages } = await A.listMessages(cid, N.stream, { startSeq: String(seq), limit: '1', maxPayloadBytes: 16 })
    const m = messages.find((x) => Number(x.sequence) === seq)!
    expect(m.truncated).toBe(true)
    expect(Number(m.dataSize)).toBeGreaterThan(16)
    // Full fetch returns the complete payload.
    const full = await A.getMessage(cid, N.stream, seq)
    expect(A.msgText(full.message ?? {}).length).toBeGreaterThan(1900)
  })
})

test.describe('messages: get', () => {
  test('CHK GetMessage by sequence returns that message', async () => {
    const got = await A.getMessage(cid, N.stream, 1)
    expect(Number(got.message?.sequence)).toBe(1)
  })

  test('CHK GetMessage of a missing sequence is not_found', async () => {
    try {
      await A.getMessage(cid, N.stream, 999999)
      throw new Error('expected not_found')
    } catch (e) {
      expect(e).toBeInstanceOf(A.ConnectError)
      expect((e as A.ConnectError).code).toBe('not_found')
    }
  })
})
