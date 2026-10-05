import { describe, it, expect, vi, beforeEach } from 'vitest'
import { create } from '@bufbuild/protobuf'
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

  it('bounds the paused buffer by size as well as by count', async () => {
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

    const big = 'A'.repeat(1 << 20)
    for (let i = 0; i < 40; i++) {
      await vi.waitFor(() => expect(push).toBeDefined())
      const next = push
      push = undefined
      next?.({ value: { event: { case: 'batch', value: { messages: [create(NatsMessageSchema, { subject: 'orders.a', dataBase64: big })] } } }, done: false })
      await vi.waitFor(() => expect(buffered).toHaveBeenCalledTimes(i + 1))
    }

    const kept = buffered.mock.lastCall?.[0] as number
    expect(kept).toBeGreaterThan(0)
    expect(kept).toBeLessThan(40)
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
