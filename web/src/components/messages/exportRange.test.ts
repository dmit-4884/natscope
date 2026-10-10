import { describe, it, expect, vi } from 'vitest'
import type { Message } from '@/types/nats'
import { walkRange, type RangePage } from './exportRange'

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

function collect(fetchPage: (startSeq: number, signal?: AbortSignal) => Promise<RangePage>, opts: { startSeq: number; limit: number; signal?: AbortSignal; onProgress?: (n: number) => void }) {
  const seqs: number[] = []
  const pages: number[] = []
  const result = walkRange(fetchPage, {
    ...opts,
    onPage: (messages) => {
      pages.push(messages.length)
      for (const m of messages) seqs.push(m.sequence)
    },
  })
  return result.then((r) => ({ ...r, seqs, pages }))
}

describe('walkRange', () => {
  it('hands over every page of the stream when under the limit', async () => {
    const { fetchPage } = fakeStream(12, 5)
    const r = await collect(fetchPage, { startSeq: 1, limit: 1000 })
    expect(r.seqs).toEqual([1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12])
    expect(r.pages).toEqual([5, 5, 2])
    expect(r.count).toBe(12)
    expect(r.reachedEnd).toBe(true)
    expect(r.truncated).toBe(false)
    expect(r.aborted).toBe(false)
  })

  it('stops at the limit, clips the last page and marks truncated when more remain', async () => {
    const { fetchPage } = fakeStream(100, 10)
    const r = await collect(fetchPage, { startSeq: 1, limit: 25 })
    expect(r.count).toBe(25)
    expect(r.seqs[24]).toBe(25)
    expect(r.pages).toEqual([10, 10, 5])
    expect(r.reachedEnd).toBe(false)
    expect(r.truncated).toBe(true)
  })

  it('is not truncated when the end coincides with the limit', async () => {
    const { fetchPage } = fakeStream(20, 5)
    const r = await collect(fetchPage, { startSeq: 1, limit: 20 })
    expect(r.count).toBe(20)
    expect(r.reachedEnd).toBe(true)
    expect(r.truncated).toBe(false)
  })

  it('reports progress after each page, never beyond the limit', async () => {
    const { fetchPage } = fakeStream(12, 5)
    const onProgress = vi.fn()
    await collect(fetchPage, { startSeq: 1, limit: 1000, onProgress })
    expect(onProgress.mock.calls.map((c) => c[0])).toEqual([5, 10, 12])
  })

  it('honors an already-aborted signal (collects nothing)', async () => {
    const { fetchPage } = fakeStream(50, 10)
    const ctrl = new AbortController()
    ctrl.abort()
    const r = await collect(fetchPage, { startSeq: 1, limit: 1000, signal: ctrl.signal })
    expect(r.count).toBe(0)
    expect(r.aborted).toBe(true)
    expect(r.truncated).toBe(false)
  })

  it('starts paging from the given startSeq', async () => {
    const { fetchPage, calls } = fakeStream(30, 10)
    const r = await collect(fetchPage, { startSeq: 11, limit: 1000 })
    expect(calls[0]).toBe(11)
    expect(r.seqs[0]).toBe(11)
    expect(r.reachedEnd).toBe(true)
  })

  it('breaks out when has_more=true but next_seq makes no forward progress', async () => {
    let calls = 0
    const fetchPage = async (): Promise<RangePage> => {
      calls++
      return { messages: [msg(1)], has_more: true, next_seq: 1 }
    }
    const r = await collect(fetchPage, { startSeq: 1, limit: 1000 })
    expect(calls).toBe(1)
    expect(r.reachedEnd).toBe(true)
  })

  it('breaks out when has_more=true but the page is empty', async () => {
    let calls = 0
    const fetchPage = async (startSeq: number): Promise<RangePage> => {
      calls++
      return { messages: [], has_more: true, next_seq: startSeq + 1 }
    }
    const r = await collect(fetchPage, { startSeq: 1, limit: 1000 })
    expect(calls).toBe(1)
    expect(r.count).toBe(0)
    expect(r.reachedEnd).toBe(true)
  })

  it('keeps what was handed over when the abort lands between pages', async () => {
    const ctrl = new AbortController()
    let calls = 0
    const fetchPage = async (startSeq: number): Promise<RangePage> => {
      calls++
      if (calls === 2) ctrl.abort()
      return { messages: [msg(startSeq)], has_more: true, next_seq: startSeq + 1 }
    }
    const r = await collect(fetchPage, { startSeq: 1, limit: 1000, signal: ctrl.signal })
    expect(r.aborted).toBe(true)
    expect(r.count).toBeGreaterThan(0)
  })

  it('treats an abort that rejects an in-flight page fetch as a clean abort with the earlier pages kept', async () => {
    const ctrl = new AbortController()
    let calls = 0
    const fetchPage = async (startSeq: number): Promise<RangePage> => {
      calls++
      if (calls === 3) {
        ctrl.abort()
        throw new DOMException('The operation was aborted', 'AbortError')
      }
      return { messages: [msg(startSeq), msg(startSeq + 1)], has_more: true, next_seq: startSeq + 2 }
    }
    const r = await collect(fetchPage, { startSeq: 1, limit: 1000, signal: ctrl.signal })
    expect(r.aborted).toBe(true)
    expect(r.seqs).toEqual([1, 2, 3, 4])
    expect(r.count).toBe(4)
  })

  it('still throws a fetch failure that is not an abort', async () => {
    const fetchPage = async (): Promise<RangePage> => {
      throw new Error('boom')
    }
    await expect(collect(fetchPage, { startSeq: 1, limit: 10, signal: new AbortController().signal })).rejects.toThrow('boom')
  })
})
