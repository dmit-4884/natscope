import { describe, it, expect, vi, beforeEach } from 'vitest'
import { create } from '@bufbuild/protobuf'
import { GetNextMessageResponseSchema } from '../gen/services/grpc/nats/v1/messages/nats_messages_service_pb'

const getNextMessageMock = vi.hoisted(() => vi.fn())

vi.mock('./grpc/clients', () => ({
  messagesClient: { getNextMessage: getNextMessageMock },
}))

import { getNextMessage } from './messages'

describe('getNextMessage', () => {
  beforeEach(() => getNextMessageMock.mockReset())

  it('asks for the first match at or after the sequence and maps it', async () => {
    getNextMessageMock.mockResolvedValue(
      create(GetNextMessageResponseSchema, {
        message: { sequence: 7n, subject: 'orders.paid', dataBase64: 'e30=', dataSize: 2, contentType: 'json' },
      }),
    )

    const message = await getNextMessage('conn-1', 'ORDERS', 5, ['orders.paid', 'orders.*'])

    expect(getNextMessageMock).toHaveBeenCalledWith(
      { connectionId: 'conn-1', streamName: 'ORDERS', startSeq: 5n, subjects: ['orders.paid', 'orders.*'] },
      { signal: undefined },
    )
    expect(message).toMatchObject({ sequence: 7, subject: 'orders.paid', content_type: 'json' })
  })

  it('is null when nothing matches', async () => {
    getNextMessageMock.mockResolvedValue(create(GetNextMessageResponseSchema, {}))

    expect(await getNextMessage('conn-1', 'ORDERS', 99, [])).toBeNull()
  })
})
