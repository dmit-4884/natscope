import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, renderHook } from '@testing-library/react'
import type { WSBatchPayload, WSErrorPayload, WSMessagePayload, WSStatsPayload } from '@/contexts/live'
import { useLiveSubscription } from './useLiveSubscription'

const { clients, setStats } = vi.hoisted(() => ({ clients: [] as FakeClient[], setStats: vi.fn() }))

interface FakeClient {
  deliver: (payload: WSBatchPayload) => void
  onError?: (payload: WSErrorPayload) => void
  onStats?: (payload: WSStatsPayload) => void
  onBuffered?: (count: number) => void
  subscribe: ReturnType<typeof vi.fn>
  subscribeSubjects: ReturnType<typeof vi.fn>
}

vi.mock('@/contexts/live', () => {
  class LiveStreamClient {
    onConnected?: () => void
    onSubscribed?: () => void
    onBatch?: (payload: WSBatchPayload) => void
    onStats?: (payload: WSStatsPayload) => void
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
  const statsState = { setStats }
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
    expect(sequences(result)).toEqual([5, 4, 3, 2, 1])
  })

  it('flushes every batch without a display rate, newest first', () => {
    const { result } = render(0)

    deliver(1, 2, 3)
    deliver(4, 5)

    expect(sequences(result)).toEqual([5, 4, 3, 2, 1])
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

    expect(client.subscribeSubjects).toHaveBeenCalledWith(['orders.>', 'audit.*'], undefined)
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
    expect(client.subscribeSubjects).toHaveBeenLastCalledWith(['open.>'], undefined)
    expect(result.current.deniedSubjects).toEqual([])
  })

  it('passes the session limits to the subject subscription', () => {
    renderHook(() =>
      useLiveSubscription({
        connectionId: 'conn-1',
        streamName: null,
        subjects: ['>'],
        enabled: true,
        initialLimit: 100,
        subjectLimits: { maxPayloadBytes: 65536, maxDisplayRate: 5 },
      }),
    )
    expect(clients[clients.length - 1].subscribeSubjects).toHaveBeenCalledWith(['>'], { maxPayloadBytes: 65536, maxDisplayRate: 5 })
  })

  it('leaves muted subjects out of the feed and the counters', () => {
    const { result } = renderHook(() =>
      useLiveSubscription({ connectionId: 'conn-1', streamName: null, subjects: ['>'], enabled: true, initialLimit: 100, exclude: ['metrics.>'] }),
    )
    const client = clients[clients.length - 1]

    act(() => client.deliver({ messages: [core('metrics.cpu'), core('orders.new'), core('metrics.mem')], count: 3 }))

    expect(result.current.liveMessages.map((m) => m.subject)).toEqual(['orders.new'])
    expect(result.current.subjectCounts).toEqual({ 'orders.new': 1 })
  })

  it('reports messages the server skipped and messages waiting while paused', () => {
    const { result } = renderHook(() =>
      useLiveSubscription({ connectionId: 'conn-1', streamName: null, subjects: ['>'], enabled: true, initialLimit: 100, globalStats: false }),
    )
    const client = clients[clients.length - 1]

    act(() => client.onStats?.({ messages_received: 20, messages_dropped: 4, msg_per_second: 2 }))
    act(() => client.onBuffered?.(7))

    expect(result.current.messagesDropped).toBe(4)
    expect(result.current.pausedCount).toBe(7)
  })

  it('keeps its own message rate and leaves the shared stream stats alone', () => {
    setStats.mockClear()
    const { result } = renderHook(() =>
      useLiveSubscription({
        connectionId: 'conn-1',
        streamName: null,
        subjects: ['>'],
        enabled: true,
        initialLimit: 100,
        globalStats: false,
      }),
    )
    const client = clients[clients.length - 1]

    act(() => client.onStats?.({ messages_received: 10, messages_dropped: 0, msg_per_second: 7 }))

    expect(result.current.msgPerSecond).toBe(7)
    expect(setStats).not.toHaveBeenCalled()
  })
})

describe('useLiveSubscription subject sessions and the server', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    clients.length = 0
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  const core = (subject: string): WSMessagePayload => ({ stream_name: '', subject, timestamp: 1, data_base64: '', data_size: 0 })

  it('reopens the subscription with the muted subjects so the server leaves them out', () => {
    const { rerender } = renderHook(
      ({ exclude }) =>
        useLiveSubscription({
          connectionId: 'conn-1',
          streamName: null,
          subjects: ['>'],
          enabled: true,
          initialLimit: 100,
          subjectLimits: { maxDisplayRate: 10, exclude },
        }),
      { initialProps: { exclude: [] as string[] } },
    )

    rerender({ exclude: ['metrics.>'] })

    expect(clients[clients.length - 1].subscribeSubjects).toHaveBeenLastCalledWith(['>'], { maxDisplayRate: 10, exclude: ['metrics.>'] })
  })

  it('shows what the server sends at once, as the server applies the display rate', () => {
    const { result } = renderHook(() =>
      useLiveSubscription({
        connectionId: 'conn-1',
        streamName: null,
        subjects: ['>'],
        enabled: true,
        initialLimit: 100,
        maxDisplayRate: 1,
        subjectLimits: { maxDisplayRate: 1 },
      }),
    )

    act(() => clients[clients.length - 1].deliver({ messages: [core('a'), core('b'), core('c')], count: 3 }))

    expect(result.current.liveMessages).toHaveLength(3)
  })
})
