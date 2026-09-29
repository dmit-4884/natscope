import { describe, it, expect, vi, beforeEach } from 'vitest'

const publishMessageCall = vi.fn()

vi.mock('./grpc/clients', () => ({
  publishClient: { publishMessage: publishMessageCall },
}))

const { publishMessage } = await import('./publish')

beforeEach(() => {
  vi.clearAllMocks()
  publishMessageCall.mockResolvedValue({ stream: 'HITS', sequence: 3n, duplicate: false })
})

describe('publishMessage', () => {
  it('sends a JSON body as a string', async () => {
    await publishMessage({ connection_id: 'conn-1', subject: 'orders.new', data: { id: 1 } })

    expect(publishMessageCall.mock.calls[0][0].data).toBe('{"id":1}')
  })

  it('sends no body for a counter increment', async () => {
    await publishMessage({
      connection_id: 'conn-1',
      subject: 'hits.page',
      data: null,
      headers: { 'Nats-Incr': '+1' },
    })

    expect(publishMessageCall.mock.calls[0][0].data).toBe('')
    expect(publishMessageCall.mock.calls[0][0].headers).toEqual({ 'Nats-Incr': '+1' })
  })

  it('returns the counter total', async () => {
    publishMessageCall.mockResolvedValue({ stream: 'HITS', sequence: 3n, duplicate: false, counterValue: '12' })

    const res = await publishMessage({ connection_id: 'conn-1', subject: 'hits.page', data: null })

    expect(res.counter_value).toBe('12')
  })

  it('omits the counter total for plain messages', async () => {
    const res = await publishMessage({ connection_id: 'conn-1', subject: 'orders.new', data: {} })

    expect(res.counter_value).toBeUndefined()
  })
})
