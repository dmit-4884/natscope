import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, act, waitFor } from '@testing-library/react'
import * as messagesApi from '@/api/messages'
import type { MessagesResponse } from '@/api/messages'
import type { Message } from '@/types/nats'
import { useLoadMoreMessages } from './useLoadMoreMessages'

vi.mock('@/api/messages', () => ({
  getMessages: vi.fn(),
}))

vi.mock('@/utils/toast', () => ({
  toast: { error: vi.fn(), success: vi.fn(), warning: vi.fn() },
}))

const api = vi.mocked(messagesApi)

function msg(sequence: number): Message {
  return {
    sequence,
    subject: `orders.${sequence}`,
    timestamp: sequence * 1000,
    data_base64: '',
    data_size: 0,
    content_type: 'json',
  }
}

function page(seqs: number[], hasMore: boolean, nextSeq: number): MessagesResponse {
  return { messages: seqs.map(msg), has_more: hasMore, next_seq: nextSeq }
}

const baseOpts = {
  streamName: 'ORDERS',
  connectionId: 'conn-1',
  limit: 2,
  direction: 'backward' as const,
}

describe('useLoadMoreMessages', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('returns base page messages and hasMore as-is', () => {
    const base = page([10, 9], true, 8)
    const { result } = renderHook(() => useLoadMoreMessages({ ...baseOpts, baseData: base }))
    expect(result.current.messages.map((m) => m.sequence)).toEqual([10, 9])
    expect(result.current.hasMore).toBe(true)
    expect(result.current.isLoadingMore).toBe(false)
  })

  it('appends the next page fetched from next_seq', async () => {
    api.getMessages.mockResolvedValueOnce(page([8, 7], false, 0))
    const base = page([10, 9], true, 8)
    const { result } = renderHook(() => useLoadMoreMessages({ ...baseOpts, baseData: base }))

    await act(() => result.current.loadMore())

    expect(api.getMessages).toHaveBeenCalledWith('ORDERS', {
      connection_id: 'conn-1',
      start_seq: 8,
      limit: 2,
      subject_filter: undefined,
      content_filter: undefined,
      direction: 'backward',
    })
    expect(result.current.messages.map((m) => m.sequence)).toEqual([10, 9, 8, 7])
    expect(result.current.hasMore).toBe(false)
  })

  it('deduplicates overlapping sequences across pages', async () => {
    api.getMessages.mockResolvedValueOnce(page([9, 8], false, 0))
    const base = page([10, 9], true, 9)
    const { result } = renderHook(() => useLoadMoreMessages({ ...baseOpts, baseData: base }))

    await act(() => result.current.loadMore())

    expect(result.current.messages.map((m) => m.sequence)).toEqual([10, 9, 8])
  })

  it('resets accumulated pages when the base page changes', async () => {
    api.getMessages.mockResolvedValueOnce(page([8, 7], true, 6))
    const initial = page([10, 9], true, 8)
    const { result, rerender } = renderHook(
      ({ baseData }: { baseData: MessagesResponse }) =>
        useLoadMoreMessages({ ...baseOpts, baseData }),
      { initialProps: { baseData: initial } },
    )

    await act(() => result.current.loadMore())
    expect(result.current.messages).toHaveLength(4)

    rerender({ baseData: page([20, 19], true, 18) })
    await waitFor(() =>
      expect(result.current.messages.map((m) => m.sequence)).toEqual([20, 19]),
    )
  })

  it('discards a late page that resolves after the base changed', async () => {
    let resolve!: (value: MessagesResponse) => void
    api.getMessages.mockImplementationOnce(
      () => new Promise<MessagesResponse>((r) => (resolve = r)),
    )
    const { result, rerender } = renderHook(
      ({ baseData }: { baseData: MessagesResponse }) =>
        useLoadMoreMessages({ ...baseOpts, baseData }),
      { initialProps: { baseData: page([10, 9], true, 8) } },
    )

    let pending!: Promise<void>
    act(() => {
      pending = result.current.loadMore()
    })
    rerender({ baseData: page([20, 19], true, 18) })
    await act(async () => {
      resolve(page([8, 7], true, 6))
      await pending
    })

    expect(result.current.messages.map((m) => m.sequence)).toEqual([20, 19])
    expect(result.current.hasMore).toBe(true)
  })

  it('ignores loadMore when there is nothing more to fetch', async () => {
    const base = page([10, 9], false, 0)
    const { result } = renderHook(() => useLoadMoreMessages({ ...baseOpts, baseData: base }))
    await act(() => result.current.loadMore())
    expect(api.getMessages).not.toHaveBeenCalled()
  })
})
