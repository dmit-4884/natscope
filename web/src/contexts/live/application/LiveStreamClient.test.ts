import { describe, it, expect, vi, beforeEach } from 'vitest'
import { create } from '@bufbuild/protobuf'
import { Code, ConnectError } from '@connectrpc/connect'
import { NatsMessageSchema } from '@/gen/types/nats/nats_message_pb'

const subscribeCall = vi.hoisted(() => vi.fn())

vi.mock('@/api/grpc/clients', () => ({
  liveClient: { subscribe: subscribeCall },
}))

const { LiveStreamClient } = await import('./LiveStreamClient')

function openStream(_req: unknown, opts: { signal: AbortSignal }): AsyncIterable<never> {
  return {
    [Symbol.asyncIterator]: () => ({
      next: () =>
        new Promise((_resolve, reject) => {
          opts.signal.addEventListener('abort', () => reject(new DOMException('aborted', 'AbortError')))
        }),
    }),
  }
}

describe('LiveStreamClient', () => {
  beforeEach(() => {
    subscribeCall.mockReset()
    subscribeCall.mockImplementation(openStream)
  })

  it('opens one stream per subscription', () => {
    const client = new LiveStreamClient('conn-1')
    client.connect()
    client.subscribeSubjects(['orders.>'])
    client.subscribeSubjects(['orders.>'])

    expect(subscribeCall).toHaveBeenCalledTimes(1)
    expect(subscribeCall.mock.calls[0][0]).toEqual({ connectionId: 'conn-1', subscriptions: [{ subject: 'orders.>' }] })
    client.disconnect()
  })

  it('sends the subject session limits and reopens when they change', () => {
    const client = new LiveStreamClient('conn-1')
    client.connect()
    client.subscribeSubjects(['orders.>'], { maxPayloadBytes: 65536, maxDisplayRate: 10 })
    client.subscribeSubjects(['orders.>'], { maxPayloadBytes: 65536, maxDisplayRate: 10 })
    client.subscribeSubjects(['orders.>'], { maxPayloadBytes: 65536, maxDisplayRate: 0 })

    expect(subscribeCall).toHaveBeenCalledTimes(2)
    expect(subscribeCall.mock.calls[0][0]).toEqual({
      connectionId: 'conn-1',
      subscriptions: [{ subject: 'orders.>' }],
      maxPayloadBytes: 65536,
      maxDisplayRate: 10,
    })
    expect(subscribeCall.mock.calls[1][0]).toMatchObject({ maxDisplayRate: 0 })
    client.disconnect()
  })

  it('sends the muted subjects to the server and reopens when they change', () => {
    const client = new LiveStreamClient('conn-1')
    client.connect()
    client.subscribeSubjects(['>'], { maxDisplayRate: 10, exclude: ['metrics.>'] })
    client.subscribeSubjects(['>'], { maxDisplayRate: 10, exclude: ['metrics.>'] })
    client.subscribeSubjects(['>'], { maxDisplayRate: 10, exclude: [] })

    expect(subscribeCall).toHaveBeenCalledTimes(2)
    expect(subscribeCall.mock.calls[0][0]).toMatchObject({ excludeSubjects: ['metrics.>'] })
    expect(subscribeCall.mock.calls[1][0]).toMatchObject({ excludeSubjects: [] })
    client.disconnect()
  })

  it('reports how many messages wait while paused', async () => {
    let push: ((value: IteratorResult<unknown>) => void) | undefined
    subscribeCall.mockImplementation(() => ({
      [Symbol.asyncIterator]: () => ({
        next: () => new Promise((resolve) => (push = resolve)),
      }),
    }))
    const client = new LiveStreamClient('conn-1')
    const buffered = vi.fn()
    client.onBuffered = buffered
    client.connect()
    client.subscribeSubjects(['orders.>'])
    client.pause()

    const messages = [create(NatsMessageSchema, { subject: 'orders.a' }), create(NatsMessageSchema, { subject: 'orders.b' })]
    const batch = { event: { case: 'batch', value: { messages } } }
    push?.({ value: batch, done: false })
    await vi.waitFor(() => expect(buffered).toHaveBeenLastCalledWith(2))

    client.resume()
    expect(buffered).toHaveBeenLastCalledWith(0)
    client.disconnect()
  })

  it('drops what waits while paused when told to', async () => {
    let push: ((value: IteratorResult<unknown>) => void) | undefined
    subscribeCall.mockImplementation(() => ({
      [Symbol.asyncIterator]: () => ({
        next: () => new Promise((resolve) => (push = resolve)),
      }),
    }))
    const client = new LiveStreamClient('conn-1')
    const buffered = vi.fn()
    const batches = vi.fn()
    client.onBuffered = buffered
    client.onBatch = batches
    client.connect()
    client.subscribeSubjects(['orders.>'])
    client.pause()

    push?.({ value: { event: { case: 'batch', value: { messages: [create(NatsMessageSchema, { subject: 'orders.a' })] } } }, done: false })
    await vi.waitFor(() => expect(buffered).toHaveBeenLastCalledWith(1))

    client.discardPaused()
    expect(buffered).toHaveBeenLastCalledWith(0)
    client.resume()
    expect(batches).not.toHaveBeenCalled()
    client.disconnect()
  })

  it('bounds the paused buffer by size as well as by count', async () => {
    let push: ((value: IteratorResult<unknown>) => void) | undefined
    subscribeCall.mockImplementation(() => ({
      [Symbol.asyncIterator]: () => ({
        next: () => new Promise((resolve) => (push = resolve)),
      }),
    }))
    const client = new LiveStreamClient('conn-1')
    const replayed = vi.fn()
    client.onBatch = replayed
    client.connect()
    client.subscribeSubjects(['orders.>'])
    client.pause()

    const big = 'A'.repeat(1 << 20)
    for (let i = 0; i < 40; i++) {
      await vi.waitFor(() => expect(push).toBeDefined())
      const next = push
      push = undefined
      next?.({ value: { event: { case: 'batch', value: { messages: [create(NatsMessageSchema, { subject: 'orders.a', dataBase64: big })] } } }, done: false })
    }
    await vi.waitFor(() => expect(push).toBeDefined())

    client.resume()
    const kept = replayed.mock.calls.reduce((total, [batch]) => total + (batch as { count: number }).count, 0)
    expect(kept).toBeGreaterThan(0)
    expect(kept).toBeLessThan(40)
    client.disconnect()
  })

  it('counts every message that arrived while paused, not only the ones it keeps', async () => {
    let push: ((value: IteratorResult<unknown>) => void) | undefined
    subscribeCall.mockImplementation(() => ({
      [Symbol.asyncIterator]: () => ({
        next: () => new Promise((resolve) => (push = resolve)),
      }),
    }))
    const client = new LiveStreamClient('conn-1')
    const buffered = vi.fn()
    client.onBuffered = buffered
    client.connect()
    client.subscribeSubjects(['tel.>'])
    client.pause()

    const messages = Array.from({ length: 3000 }, () => create(NatsMessageSchema, { subject: 'tel.eu.1' }))
    for (let i = 0; i < 3; i++) {
      await vi.waitFor(() => expect(push).toBeDefined())
      const next = push
      push = undefined
      next?.({ value: { event: { case: 'batch', value: { messages } } }, done: false })
    }

    await vi.waitFor(() => expect(buffered).toHaveBeenLastCalledWith(9000), { timeout: 3000 })
    expect(buffered.mock.calls.map(([n]) => n)).toEqual([...buffered.mock.calls.map(([n]) => n)].sort((a, b) => a - b))
    client.disconnect()
  })

  it('reports the paused count at most about once a second', async () => {
    let push: ((value: IteratorResult<unknown>) => void) | undefined
    subscribeCall.mockImplementation(() => ({
      [Symbol.asyncIterator]: () => ({
        next: () => new Promise((resolve) => (push = resolve)),
      }),
    }))
    const client = new LiveStreamClient('conn-1')
    const buffered = vi.fn()
    client.onBuffered = buffered
    client.connect()
    client.subscribeSubjects(['tel.>'])
    client.pause()

    for (let i = 0; i < 20; i++) {
      await vi.waitFor(() => expect(push).toBeDefined())
      const next = push
      push = undefined
      next?.({ value: { event: { case: 'batch', value: { messages: [create(NatsMessageSchema, { subject: 'tel.eu.1' })] } } }, done: false })
    }
    await vi.waitFor(() => expect(push).toBeDefined())

    expect(buffered.mock.calls.length).toBeLessThanOrEqual(2)
    client.disconnect()
  })

  it('reconnects when the server aborts a stalled stream', async () => {
    vi.useFakeTimers()
    try {
      subscribeCall.mockImplementationOnce(() => ({
        [Symbol.asyncIterator]: () => ({ next: () => Promise.reject(new ConnectError('live consumer stalled', Code.Aborted)) }),
      }))
      const client = new LiveStreamClient('conn-1')
      const reconnecting = vi.fn()
      const disconnected = vi.fn()
      client.onReconnecting = reconnecting
      client.onDisconnect = disconnected
      client.connect()
      client.subscribeSubjects(['orders.>'])

      await vi.waitFor(() => expect(reconnecting).toHaveBeenCalled())
      await vi.advanceTimersByTimeAsync(5000)
      expect(subscribeCall).toHaveBeenCalledTimes(2)
      expect(disconnected).not.toHaveBeenCalled()
      client.disconnect()
    } finally {
      vi.useRealTimers()
    }
  })

  it('gives up at once on a subject the server rejects', async () => {
    subscribeCall.mockImplementationOnce(() => ({
      [Symbol.asyncIterator]: () => ({ next: () => Promise.reject(new ConnectError('invalid subject', Code.InvalidArgument)) }),
    }))
    const client = new LiveStreamClient('conn-1')
    const reconnecting = vi.fn()
    const disconnected = vi.fn()
    client.onReconnecting = reconnecting
    client.onDisconnect = disconnected
    client.connect()
    client.subscribeSubjects(['orders.\u0001'])

    await vi.waitFor(() => expect(disconnected).toHaveBeenCalled())
    expect(reconnecting).not.toHaveBeenCalled()
  })

  it('reports the NATS connection going down and back', async () => {
    let push: ((value: IteratorResult<unknown>) => void) | undefined
    subscribeCall.mockImplementation(() => ({
      [Symbol.asyncIterator]: () => ({
        next: () => new Promise((resolve) => (push = resolve)),
      }),
    }))
    const client = new LiveStreamClient('conn-1')
    const link = vi.fn()
    client.onLink = link
    client.connect()
    client.subscribeSubjects(['orders.>'])

    await vi.waitFor(() => expect(push).toBeDefined())
    push?.({ value: { event: { case: 'connection', value: { connected: false } } }, done: false })
    await vi.waitFor(() => expect(link).toHaveBeenLastCalledWith(false))
    push?.({ value: { event: { case: 'connection', value: { connected: true } } }, done: false })
    await vi.waitFor(() => expect(link).toHaveBeenLastCalledWith(true))
    client.disconnect()
  })

  it('opens no stream once disconnected', () => {
    const client = new LiveStreamClient('conn-1')
    client.connect()
    client.disconnect()

    client.subscribeSubjects(['orders.>'])
    client.subscribe('ORDERS')

    expect(subscribeCall).not.toHaveBeenCalled()
  })
})
