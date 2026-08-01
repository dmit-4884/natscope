import { describe, it, expect, vi, beforeEach } from 'vitest'
import { create } from '@bufbuild/protobuf'
import { DurationSchema } from '@bufbuild/protobuf/wkt'

import { ConsumerInfoSchema } from '../gen/types/nats/nats_stream_pb'

const createConsumerCall = vi.fn()
const updateConsumerCall = vi.fn()
const putKVKeyCall = vi.fn()

vi.mock('./grpc/clients', () => ({
  streamsClient: {},
  managementClient: {
    createConsumer: createConsumerCall,
    updateConsumer: updateConsumerCall,
    putKVKey: putKVKeyCall,
  },
}))

const { createConsumer, updateConsumer, putKVKey } = await import('./management')

beforeEach(() => {
  vi.clearAllMocks()
  createConsumerCall.mockResolvedValue({ consumer: create(ConsumerInfoSchema, { name: 'worker' }) })
  updateConsumerCall.mockResolvedValue({ consumer: create(ConsumerInfoSchema, { name: 'worker' }) })
  putKVKeyCall.mockResolvedValue({ revision: 7n })
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

  it('sends the ephemeral flag when the caller asks for a non-durable consumer', async () => {
    await createConsumer('conn-1', 'ORDERS', { name: 'worker', ephemeral: true })

    expect(createConsumerCall.mock.calls[0][0].ephemeral).toBe(true)
  })

  it('defaults ephemeral to false so consumers stay durable', async () => {
    await createConsumer('conn-1', 'ORDERS', { name: 'worker' })

    expect(createConsumerCall.mock.calls[0][0].ephemeral).toBe(false)
  })

  it('still sends the filters supplied at creation time', async () => {
    await createConsumer('conn-1', 'ORDERS', {
      name: 'worker',
      filter_subjects: ['orders.created', 'orders.paid'],
    })

    expect(createConsumerCall.mock.calls[0][0].filterSubjects).toEqual(['orders.created', 'orders.paid'])
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

  it('omits filterSubject when the caller does not provide one', async () => {
    await updateConsumer('conn-1', 'ORDERS', 'worker', { description: 'same filters' })

    expect(updateConsumerCall.mock.calls[0][0].filterSubject).toBeUndefined()
    expect(updateConsumerCall.mock.calls[0][0].filterSubjects).toEqual([])
  })

  it('sends a provided filterSubject so the server replaces the filter', async () => {
    await updateConsumer('conn-1', 'ORDERS', 'worker', { filter_subject: 'orders.paid' })

    expect(updateConsumerCall.mock.calls[0][0].filterSubject).toBe('orders.paid')
  })

  it('sends an empty filterSubject so the server clears the filter', async () => {
    await updateConsumer('conn-1', 'ORDERS', 'worker', { filter_subject: '' })

    expect(updateConsumerCall.mock.calls[0][0].filterSubject).toBe('')
  })

  it('sends filterSubjects so the server replaces the whole list', async () => {
    await updateConsumer('conn-1', 'ORDERS', 'worker', {
      filter_subjects: ['orders.created', 'orders.paid'],
    })

    expect(updateConsumerCall.mock.calls[0][0].filterSubjects).toEqual(['orders.created', 'orders.paid'])
    expect(updateConsumerCall.mock.calls[0][0].filterSubject).toBeUndefined()
  })

  it('treats an empty filterSubjects list as "keep the existing filters"', async () => {
    await updateConsumer('conn-1', 'ORDERS', 'worker', { filter_subjects: [] })

    expect(updateConsumerCall.mock.calls[0][0].filterSubjects).toEqual([])
  })
})

describe('putKVKey', () => {
  it('sends the loaded revision so a stale edit is rejected', async () => {
    const result = await putKVKey('conn-1', 'config', 'greeting', 'hello', 4)

    expect(putKVKeyCall.mock.calls[0][0].revision).toBe(4n)
    expect(result).toEqual({ revision: 7 })
  })

  it('sends no expected revision when creating a key', async () => {
    await putKVKey('conn-1', 'config', 'greeting', 'hello')

    expect(putKVKeyCall.mock.calls[0][0].revision).toBe(0n)
  })
})
