import { describe, it, expect, vi, beforeEach } from 'vitest'
import { create } from '@bufbuild/protobuf'
import { DurationSchema, TimestampSchema } from '@bufbuild/protobuf/wkt'
import { ListServicesResponseSchema } from '../gen/services/grpc/nats/v1/discovery/nats_discovery_service_pb'
import { AccessStatus } from '../gen/types/nats/nats_access_pb'

const listServicesMock = vi.hoisted(() => vi.fn())

vi.mock('./grpc/clients', () => ({
  discoveryClient: { listServices: listServicesMock },
}))

import { listServices } from './discovery'

describe('listServices', () => {
  beforeEach(() => listServicesMock.mockReset())

  it('maps services, instances, endpoint stats and proto matches', async () => {
    const endpoint = {
      name: 'Create',
      subject: 'orders.create',
      queueGroup: 'q',
      metadata: { k: 'v' },
      stats: {
        numRequests: 4n,
        numErrors: 1n,
        lastError: 'boom',
        processingTime: create(DurationSchema, { nanos: 40_000_000 }),
        averageProcessingTime: create(DurationSchema, { nanos: 10_000_000 }),
      },
      protoMethod: { sourceId: 'src', service: 'shop.OrderService', method: 'Create', inputType: 'shop.In', outputType: 'shop.Out' },
    }
    listServicesMock.mockResolvedValue(
      create(ListServicesResponseSchema, {
        infoAccess: { status: AccessStatus.ALLOWED, operation: 'publish', subject: '$SRV.INFO' },
        statsAccess: { status: AccessStatus.DENIED, operation: 'publish', subject: '$SRV.STATS' },
        services: [
          {
            name: 'orders',
            description: 'Order API',
            versions: ['1.0.0'],
            instances: [
              { id: 'a', version: '1.0.0', started: create(TimestampSchema, { seconds: 1_790_000_000n }), endpoints: [endpoint] },
              { id: 'b', version: '1.0.0', endpoints: [{ name: 'Create', subject: 'orders.create' }] },
            ],
            endpoints: [endpoint],
          },
        ],
      }),
    )

    const got = await listServices('conn-1')

    expect(listServicesMock).toHaveBeenCalledWith({ connectionId: 'conn-1' }, { signal: undefined })
    expect(got.info_access).toEqual({ status: 'allowed', operation: 'publish', subject: '$SRV.INFO' })
    expect(got.stats_access).toEqual({ status: 'denied', operation: 'publish', subject: '$SRV.STATS' })
    expect(got.services).toHaveLength(1)
    const svc = got.services[0]
    expect(svc.instances[0].started).toBe(1_790_000_000_000)
    expect(svc.instances[1].started).toBeUndefined()
    expect(svc.instances[1].endpoints[0].stats).toBeUndefined()
    expect(svc.endpoints[0]).toEqual({
      name: 'Create',
      subject: 'orders.create',
      queue_group: 'q',
      metadata: { k: 'v' },
      stats: {
        num_requests: 4,
        num_errors: 1,
        last_error: 'boom',
        processing_time_ns: 40_000_000,
        average_processing_time_ns: 10_000_000,
      },
      proto_method: { source_id: 'src', service: 'shop.OrderService', method: 'Create', input_type: 'shop.In', output_type: 'shop.Out' },
    })
  })

  it('leaves stats access unset when info access was denied', async () => {
    listServicesMock.mockResolvedValue(
      create(ListServicesResponseSchema, {
        infoAccess: { status: AccessStatus.DENIED, operation: 'subscribe', subject: '_INBOX.x' },
      }),
    )

    const got = await listServices('conn-1')

    expect(got.info_access).toEqual({ status: 'denied', operation: 'subscribe', subject: '_INBOX.x' })
    expect(got.stats_access).toBeUndefined()
    expect(got.services).toEqual([])
  })
})
