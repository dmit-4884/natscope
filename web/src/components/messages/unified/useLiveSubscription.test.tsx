import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, renderHook } from '@testing-library/react'
import type { WSBatchPayload, WSMessagePayload } from '@/contexts/live'
import { useLiveSubscription } from './useLiveSubscription'

const { clients } = vi.hoisted(() => ({ clients: [] as FakeClient[] }))

interface FakeClient {
  deliver: (payload: WSBatchPayload) => void
}

vi.mock('@/contexts/live', () => {
  class LiveStreamClient {
    onConnected?: () => void
    onSubscribed?: () => void
    onBatch?: (payload: WSBatchPayload) => void
    onStats?: () => void
    onError?: () => void
    onDisconnect?: () => void
    onReconnecting?: () => void
    paused: WSBatchPayload[] | null = null
    subscribe = vi.fn()
    unsubscribe = vi.fn()
    disconnect = vi.fn()
    constructor() {
      clients.push(this)
    }
    connect() {
      this.onConnected?.()
    }
    pause() {
      this.paused = []
    }
    resume() {
      const buffered = this.paused ?? []
      this.paused = null
      buffered.forEach((b) => this.onBatch?.(b))
    }
    deliver(payload: WSBatchPayload) {
      if (this.paused) this.paused.push(payload)
      else this.onBatch?.(payload)
    }
  }
  const statsState = { setStats: () => {} }
  return {
    LiveStreamClient,
    useLiveStatsStore: (selector: (s: typeof statsState) => unknown) => selector(statsState),
    useProtoReloadInvalidation: () => {},
  }
})

const STREAM = 'ORDERS'

function message(seq: number): WSMessagePayload {
  return { stream_name: STREAM, subject: `ORDERS.${seq}`, sequence: seq, timestamp: seq, data_base64: '', data_size: 0 }
}

function deliver(...seqs: number[]) {
  act(() => clients[clients.length - 1]?.deliver({ messages: seqs.map(message), count: seqs.length }))
}

function render(maxDisplayRate: number) {
  return renderHook(() =>
    useLiveSubscription({ connectionId: 'conn-1', streamName: STREAM, enabled: true, maxDisplayRate, initialLimit: 100 }),
  )
}

const sequences = (result: { current: ReturnType<typeof useLiveSubscription> }) =>
  result.current.liveMessages.map((m) => m.sequence)

describe('useLiveSubscription display rate', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    clients.length = 0
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('releases a burst one message at a time at the configured rate', () => {
    const { result } = render(5)

    deliver(1, 2, 3)
    expect(sequences(result)).toEqual([1])

    act(() => vi.advanceTimersByTime(199))
    expect(sequences(result)).toEqual([1])

    act(() => vi.advanceTimersByTime(1))
    expect(sequences(result)).toEqual([2, 1])

    act(() => vi.advanceTimersByTime(200))
    expect(sequences(result)).toEqual([3, 2, 1])
  })

  it('spaces a message that arrives right after the previous one', () => {
    const { result } = render(5)

    deliver(1)
    act(() => vi.advanceTimersByTime(50))
    deliver(2)
    expect(sequences(result)).toEqual([1])

    act(() => vi.advanceTimersByTime(150))
    expect(sequences(result)).toEqual([2, 1])
  })

  it('keeps at most one second of backlog, dropping the oldest', () => {
    const { result } = render(2)

    deliver(1, 2, 3, 4, 5)
    act(() => vi.advanceTimersByTime(2000))

    expect(sequences(result)).toEqual([5, 4])
  })

  it('shows queued messages at once when paused and replays the paused buffer on resume', () => {
    const { result } = render(5)

    deliver(1, 2, 3)
    act(() => result.current.togglePause())
    expect(sequences(result)).toEqual([3, 2, 1])

    deliver(4, 5)
    act(() => vi.advanceTimersByTime(1000))
    expect(sequences(result)).toEqual([3, 2, 1])

    act(() => result.current.togglePause())
    expect(sequences(result)).toEqual([4, 5, 3, 2, 1])
  })

  it('flushes every batch without a display rate', () => {
    const { result } = render(0)

    deliver(1, 2, 3)

    expect(sequences(result)).toEqual([1, 2, 3])
  })
})
