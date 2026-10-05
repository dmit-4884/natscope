import { describe, it, expect, vi, beforeEach } from 'vitest'
import { create } from '@bufbuild/protobuf'
import { GetAllConsumersResponseSchema } from '../gen/services/grpc/nats/v1/stats/nats_stats_service_pb'
import { AccessStatus } from '../gen/types/nats/nats_access_pb'

const getAllConsumersMock = vi.hoisted(() => vi.fn())

vi.mock('./grpc/clients', () => ({
  statsClient: { getAllConsumers: getAllConsumersMock },
}))

import { getConsumersOverview } from './stats'

describe('getConsumersOverview', () => {
  beforeEach(() => getAllConsumersMock.mockReset())

  it('maps consumers, their streams and the streams that could not be read', async () => {
    getAllConsumersMock.mockResolvedValue(
      create(GetAllConsumersResponseSchema, {
        consumers: [{ name: 'worker', stream: 'ORDERS', numPending: 4n, numAckPending: 2, config: { maxAckPending: 10 } }],
        streams: [{ config: { name: 'ORDERS', maxMsgs: 100n }, state: { msgs: 95n } }],
        unreadableStreams: [
          { stream: 'SECRET', access: { status: AccessStatus.DENIED, operation: 'publish', subject: '$JS.API.CONSUMER.LIST.SECRET' } },
          { stream: 'BROKEN', error: 'stream offline' },
        ],
      }),
    )

    const overview = await getConsumersOverview('conn-1')

    expect(getAllConsumersMock).toHaveBeenCalledWith({ connectionId: 'conn-1' }, { signal: undefined })
    expect(overview.consumers).toHaveLength(1)
    expect(overview.consumers[0]).toMatchObject({ name: 'worker', stream_name: 'ORDERS', num_pending: 4, num_ack_pending: 2 })
    expect(overview.consumers[0].config?.max_ack_pending).toBe(10)
    expect(overview.streams).toHaveLength(1)
    expect(overview.streams[0]).toMatchObject({ name: 'ORDERS', messages: 95 })
    expect(overview.streams[0].config.max_msgs).toBe(100)
    expect(overview.unreadable).toEqual([
      { stream: 'SECRET', access: { status: 'denied', operation: 'publish', subject: '$JS.API.CONSUMER.LIST.SECRET' }, error: undefined },
      { stream: 'BROKEN', access: undefined, error: 'stream offline' },
    ])
  })
})
