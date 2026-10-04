import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, renderHook } from '@testing-library/react'
import type { WSBatchPayload, WSErrorPayload, WSMessagePayload } from '@/contexts/live'
import { useLiveSubscription } from './useLiveSubscription'

const { clients } = vi.hoisted(() => ({ clients: [] as FakeClient[] }))

interface FakeClient {
  deliver: (payload: WSBatchPayload) => void
  onError?: (payload: WSErrorPayload) => void
  subscribe: ReturnType<typeof vi.fn>
  subscribeSubjects: ReturnType<typeof vi.fn>
}

vi.mock('@/contexts/live', () => {
  class LiveStreamClient {
    onConnected?: () => void
    onSubscribed?: () => void
    onBatch?: (payload: WSBatchPayload) => void
    onStats?: () => void
    onError?: (payload: WSErrorPayload) => void
    onDisconnect?: () => void
    onReconnecting?: () => void
    paused: WSBatchPayload[] | null = null
    subscribe = vi.fn()
    subscribeSubjects = vi.fn()
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

describe('useLiveSubscription subjects', () => {
  beforeEach(() => {
    clients.length = 0
  })

  function renderSubjects(subjects: string[]) {
    return renderHook(
      ({ list }) =>
        useLiveSubscription({ connectionId: 'conn-1', streamName: null, subjects: list, enabled: true, initialLimit: 100 }),
      { initialProps: { list: subjects } },
    )
  }

  const core = (subject: string): WSMessagePayload => ({
    stream_name: '',
    subject,
    timestamp: 1,
    data_base64: '',
    data_size: 0,
    reply: '_INBOX.1',
  })

  it('subscribes to the subjects and keeps messages from no stream', () => {
    const { result } = renderSubjects(['orders.>', 'audit.*'])
    const client = clients[clients.length - 1]

    expect(client.subscribeSubjects).toHaveBeenCalledWith(['orders.>', 'audit.*'])
    expect(client.subscribe).not.toHaveBeenCalled()

    act(() => client.deliver({ messages: [core('orders.new')], count: 1 }))
    expect(result.current.liveMessages.map((m) => m.subject)).toEqual(['orders.new'])
    expect(result.current.liveMessages[0].reply).toBe('_INBOX.1')
  })

  it('counts every received message per subject', () => {
    const { result } = renderSubjects(['>'])
    const client = clients[clients.length - 1]

    act(() => client.deliver({ messages: [core('a'), core('b'), core('a')], count: 3 }))

    expect(result.current.subjectCounts).toEqual({ a: 2, b: 1 })
  })

  it('collects denied subjects instead of raising an error, and forgets them on a new subscription', () => {
    const { result, rerender } = renderSubjects(['open.>', 'secret.>'])
    const client = clients[clients.length - 1]

    act(() =>
      client.onError?.({
        message: 'no permission to subscribe to "secret.>"',
        code: 'SUBSCRIBE_PERMISSION_DENIED',
        access: { status: 'denied', operation: 'subscribe', subject: 'secret.>' },
      }),
    )
    expect(result.current.deniedSubjects).toEqual(['secret.>'])
    expect(result.current.wsError).toBeNull()

    rerender({ list: ['open.>'] })
    expect(client.subscribeSubjects).toHaveBeenLastCalledWith(['open.>'])
    expect(result.current.deniedSubjects).toEqual([])
  })
})
