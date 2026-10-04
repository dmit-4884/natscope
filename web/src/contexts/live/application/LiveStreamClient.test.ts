import { describe, it, expect, vi, beforeEach } from 'vitest'

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

  it('opens no stream once disconnected', () => {
    const client = new LiveStreamClient('conn-1')
    client.connect()
    client.disconnect()

    client.subscribeSubjects(['orders.>'])
    client.subscribe('ORDERS')

    expect(subscribeCall).not.toHaveBeenCalled()
  })
})
