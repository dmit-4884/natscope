import { describe, it, expect, vi, beforeEach } from 'vitest'
import { create } from '@bufbuild/protobuf'
import { DurationSchema } from '@bufbuild/protobuf/wkt'

import { ConsumerInfoSchema } from '../gen/types/nats/nats_stream_pb'

const createConsumerCall = vi.fn()
const updateConsumerCall = vi.fn()

vi.mock('./grpc/clients', () => ({
  streamsClient: {},
  managementClient: {
    createConsumer: createConsumerCall,
    updateConsumer: updateConsumerCall,
  },
}))

const { createConsumer, updateConsumer } = await import('./management')

beforeEach(() => {
  vi.clearAllMocks()
  createConsumerCall.mockResolvedValue({ consumer: create(ConsumerInfoSchema, { name: 'worker' }) })
  updateConsumerCall.mockResolvedValue({ consumer: create(ConsumerInfoSchema, { name: 'worker' }) })
})

describe('createConsumer', () => {
  it('sends backoff intervals as Durations', async () => {
    await createConsumer('conn-1', 'ORDERS', {
      name: 'worker',
      max_deliver: 5,
      backoff: [2_000_000_000, 1_500_000_000],
    })

    expect(createConsumerCall.mock.calls[0][0].backOff).toEqual([
      create(DurationSchema, { seconds: 2n, nanos: 0 }),
      create(DurationSchema, { seconds: 1n, nanos: 500_000_000 }),
    ])
  })

  it('sends an empty backoff list when the form leaves it unset', async () => {
    await createConsumer('conn-1', 'ORDERS', { name: 'worker' })

    expect(createConsumerCall.mock.calls[0][0].backOff).toEqual([])
  })
})

describe('updateConsumer', () => {
  it('sends backoff intervals as Durations', async () => {
    await updateConsumer('conn-1', 'ORDERS', 'worker', {
      max_deliver: 5,
      backoff: [2_000_000_000],
    })

    expect(updateConsumerCall.mock.calls[0][0].backOff).toEqual([
      create(DurationSchema, { seconds: 2n, nanos: 0 }),
    ])
  })

  it('sends an empty backoff list when the form leaves it unset', async () => {
    await updateConsumer('conn-1', 'ORDERS', 'worker', { max_deliver: 5 })

    expect(updateConsumerCall.mock.calls[0][0].backOff).toEqual([])
  })
})
