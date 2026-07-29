import { describe, it, expect, vi } from 'vitest'
import type { Message } from '@/types/nats'
import { collectRange, type RangePage } from './exportRange'

function msg(seq: number): Message {
  return {
    sequence: seq,
    subject: 's',
    timestamp: 0,
    data_base64: '',
    data_size: 0,
    content_type: 'json',
  }
}

// Fake stream of `total` messages (seq 1..total) served `pageSize` at a time;
// records requested pages.
function fakeStream(total: number, pageSize: number) {
  const calls: number[] = []
  const fetchPage = async (startSeq: number): Promise<RangePage> => {
    calls.push(startSeq)
    const page: Message[] = []
    for (let s = startSeq; s < startSeq + pageSize && s <= total; s++) page.push(msg(s))
    const lastSeq = page.length ? page[page.length - 1].sequence : startSeq
    const hasMore = lastSeq < total
    return { messages: page, has_more: hasMore, next_seq: hasMore ? lastSeq + 1 : 0 }
  }
  return { fetchPage, calls }
}

describe('collectRange', () => {
  it('collects the whole stream when under the limit', async () => {
    const { fetchPage } = fakeStream(12, 5)
    const r = await collectRange(fetchPage, { startSeq: 1, limit: 1000 })
    expect(r.messages.map((m) => m.sequence)).toEqual([1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12])
    expect(r.reachedEnd).toBe(true)
    expect(r.truncated).toBe(false)
    expect(r.aborted).toBe(false)
  })

  it('stops at the limit and marks truncated when more remain', async () => {
    const { fetchPage } = fakeStream(100, 10)
    const r = await collectRange(fetchPage, { startSeq: 1, limit: 25 })
    expect(r.messages).toHaveLength(25)
    expect(r.messages[0].sequence).toBe(1)
    expect(r.messages[24].sequence).toBe(25)
    expect(r.reachedEnd).toBe(false)
    expect(r.truncated).toBe(true)
  })

  it('is not truncated when the end coincides with the limit', async () => {
    const { fetchPage } = fakeStream(20, 5)
    const r = await collectRange(fetchPage, { startSeq: 1, limit: 20 })
    expect(r.messages).toHaveLength(20)
    expect(r.reachedEnd).toBe(true)
    expect(r.truncated).toBe(false)
  })

  it('reports progress after each page', async () => {
    const { fetchPage } = fakeStream(12, 5)
    const onProgress = vi.fn()
    await collectRange(fetchPage, { startSeq: 1, limit: 1000, onProgress })
    expect(onProgress.mock.calls.map((c) => c[0])).toEqual([5, 10, 12])
  })

  it('honors an already-aborted signal (collects nothing)', async () => {
    const { fetchPage } = fakeStream(50, 10)
    const ctrl = new AbortController()
    ctrl.abort()
    const r = await collectRange(fetchPage, { startSeq: 1, limit: 1000, signal: ctrl.signal })
    expect(r.messages).toHaveLength(0)
    expect(r.aborted).toBe(true)
    expect(r.truncated).toBe(false)
  })

  it('starts paging from the given startSeq', async () => {
    const { fetchPage, calls } = fakeStream(30, 10)
    const r = await collectRange(fetchPage, { startSeq: 11, limit: 1000 })
    expect(calls[0]).toBe(11)
    expect(r.messages[0].sequence).toBe(11)
    expect(r.reachedEnd).toBe(true)
  })

  // m19: no-progress guard — has_more stays true but next_seq never advances.
  it('breaks out when has_more=true but next_seq makes no forward progress', async () => {
    let calls = 0
    const fetchPage = async (_startSeq: number): Promise<RangePage> => {
      calls++
      // Buggy server that loops in place: has_more=true, next_seq stuck at
      // start.
      return { messages: [msg(1)], has_more: true, next_seq: 1 }
    }
    const r = await collectRange(fetchPage, { startSeq: 1, limit: 1000 })
    // Breaks after first page (no forward progress).
    expect(calls).toBe(1)
    expect(r.reachedEnd).toBe(true)
  })

  it('breaks out when has_more=true but the page is empty (no progress)', async () => {
    let calls = 0
    const fetchPage = async (startSeq: number): Promise<RangePage> => {
      calls++
      return { messages: [], has_more: true, next_seq: startSeq + 1 }
    }
    const r = await collectRange(fetchPage, { startSeq: 1, limit: 1000 })
    expect(calls).toBe(1)
    expect(r.messages).toHaveLength(0)
    expect(r.reachedEnd).toBe(true)
  })

  // Abort mid-export: aborted=true and messages collected before the abort are
  // returned, not discarded.
  it('aborts mid-export and returns partial results with aborted=true', async () => {
    const ctrl = new AbortController()
    let calls = 0
    const fetchPage = async (startSeq: number, _signal?: AbortSignal): Promise<RangePage> => {
      calls++
      if (calls === 2) ctrl.abort()
      const page: Message[] = [msg(startSeq)]
      return { messages: page, has_more: true, next_seq: startSeq + 1 }
    }
    const r = await collectRange(fetchPage, { startSeq: 1, limit: 1000, signal: ctrl.signal })
    expect(r.aborted).toBe(true)
    // First page collected before the abort check.
    expect(r.messages.length).toBeGreaterThan(0)
  })
})
