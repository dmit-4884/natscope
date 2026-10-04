import { describe, it, expect, vi, beforeEach } from 'vitest'

const publishMessageCall = vi.fn()

vi.mock('./grpc/clients', () => ({
  publishClient: { publishMessage: publishMessageCall },
}))

const { publishMessage, publishCoreMessage } = await import('./publish')

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

describe('publishCoreMessage', () => {
  it('sends the body as typed over core NATS', async () => {
    publishMessageCall.mockResolvedValue({ stream: '', sequence: 0n, duplicate: false })

    await publishCoreMessage({ connection_id: 'conn-1', subject: '_INBOX.x', data: 'pong', headers: { 'X-Trace': 't' } })

    expect(publishMessageCall.mock.calls[0][0]).toEqual(
      expect.objectContaining({ connectionId: 'conn-1', subject: '_INBOX.x', data: 'pong', headers: { 'X-Trace': 't' }, core: true }),
    )
  })

  it('fails with the reason the server gave', async () => {
    publishMessageCall.mockResolvedValue({ error: 'Failed to publish message: no permission to publish to "secret.op"' })

    await expect(publishCoreMessage({ connection_id: 'conn-1', subject: 'secret.op', data: '' })).rejects.toThrow(
      'no permission to publish to "secret.op"',
    )
  })
})
