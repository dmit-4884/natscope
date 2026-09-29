import { describe, it, expect, vi, beforeEach } from 'vitest'
import { create } from '@bufbuild/protobuf'
import { DurationSchema } from '@bufbuild/protobuf/wkt'

import { ConsumerInfoSchema, StreamInfoSchema } from '../gen/types/nats/nats_stream_pb'

const createConsumerCall = vi.fn()
const updateConsumerCall = vi.fn()
const putKVKeyCall = vi.fn()
const createStreamCall = vi.fn()
const updateStreamCall = vi.fn()
const resetConsumerCall = vi.fn()

vi.mock('./grpc/clients', () => ({
  streamsClient: {},
  managementClient: {
    createConsumer: createConsumerCall,
    updateConsumer: updateConsumerCall,
    putKVKey: putKVKeyCall,
    createStream: createStreamCall,
    updateStream: updateStreamCall,
    resetConsumer: resetConsumerCall,
  },
}))

const { createConsumer, updateConsumer, putKVKey, createStream, updateStream, resetConsumer } = await import('./management')

beforeEach(() => {
  vi.clearAllMocks()
  createConsumerCall.mockResolvedValue({ consumer: create(ConsumerInfoSchema, { name: 'worker' }) })
  updateConsumerCall.mockResolvedValue({ consumer: create(ConsumerInfoSchema, { name: 'worker' }) })
  putKVKeyCall.mockResolvedValue({ revision: 7n })
  const flagged = create(StreamInfoSchema, {
    config: {
      name: 'FLAGS',
      allowMsgCounter: true,
      allowMsgSchedules: true,
      subjectDeleteMarkerTtl: create(DurationSchema, { seconds: 60n }),
      persistMode: 1,
      allowBatchPublish: true,
    },
  })
  createStreamCall.mockResolvedValue({ stream: flagged })
  updateStreamCall.mockResolvedValue({ stream: flagged })
})

describe('createStream feature flags', () => {
  it('sends the NATS 2.11–2.14 stream flags', async () => {
    await createStream('conn-1', {
      name: 'FLAGS',
      allow_msg_counter: true,
      allow_msg_schedules: true,
      subject_delete_marker_ttl: 60_000_000_000,
      persist_mode: 'async',
      allow_batch_publish: true,
    })

    const req = createStreamCall.mock.calls[0][0]
    expect(req.allowMsgCounter).toBe(true)
    expect(req.allowMsgSchedules).toBe(true)
    expect(req.subjectDeleteMarkerTtl).toEqual(create(DurationSchema, { seconds: 60n, nanos: 0 }))
    expect(req.persistMode).toBe(1)
    expect(req.allowBatchPublish).toBe(true)
  })

  it('defaults the flags off and persist mode to default', async () => {
    await createStream('conn-1', { name: 'PLAIN' })

    const req = createStreamCall.mock.calls[0][0]
    expect(req.allowMsgCounter).toBe(false)
    expect(req.allowMsgSchedules).toBe(false)
    expect(req.subjectDeleteMarkerTtl).toBeUndefined()
    expect(req.persistMode).toBe(0)
    expect(req.allowBatchPublish).toBe(false)
  })

  it('maps the flags back onto the stream config', async () => {
    const info = await createStream('conn-1', { name: 'FLAGS' })

    expect(info.config.allow_msg_counter).toBe(true)
    expect(info.config.allow_msg_schedules).toBe(true)
    expect(info.config.subject_delete_marker_ttl).toBe(60_000_000_000)
    expect(info.config.persist_mode).toBe('async')
    expect(info.config.allow_batched).toBe(true)
  })
})

describe('updateStream feature flags', () => {
  it('sends only the mutable flags the caller provides', async () => {
    await updateStream('conn-1', 'FLAGS', {
      allow_msg_schedules: true,
      subject_delete_marker_ttl: 0,
      allow_batch_publish: false,
    })

    const req = updateStreamCall.mock.calls[0][0]
    expect(req.allowMsgSchedules).toBe(true)
    expect(req.subjectDeleteMarkerTtl).toEqual(create(DurationSchema, { seconds: 0n, nanos: 0 }))
    expect(req.allowBatchPublish).toBe(false)
  })

  it('leaves unset flags undefined so the server keeps them', async () => {
    await updateStream('conn-1', 'FLAGS', { description: 'x' })

    const req = updateStreamCall.mock.calls[0][0]
    expect(req.allowMsgSchedules).toBeUndefined()
    expect(req.subjectDeleteMarkerTtl).toBeUndefined()
    expect(req.allowBatchPublish).toBeUndefined()
  })
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

describe('resetConsumer', () => {
  beforeEach(() => {
    resetConsumerCall.mockResolvedValue({
      consumer: create(ConsumerInfoSchema, { name: 'worker', numPending: 3n }),
      resetSeq: 42n,
    })
  })

  it('sends the sequence as a bigint', async () => {
    await resetConsumer('conn-1', 'ORDERS', 'worker', 42)

    expect(resetConsumerCall.mock.calls[0][0]).toEqual({
      connectionId: 'conn-1',
      streamName: 'ORDERS',
      consumerName: 'worker',
      sequence: 42n,
    })
  })

  it('omits the sequence to keep the ack floor', async () => {
    await resetConsumer('conn-1', 'ORDERS', 'worker')

    expect(resetConsumerCall.mock.calls[0][0].sequence).toBeUndefined()
  })

  it('returns the reset sequence and the consumer state', async () => {
    const result = await resetConsumer('conn-1', 'ORDERS', 'worker', 42)

    expect(result.reset_seq).toBe(42)
    expect(result.consumer?.name).toBe('worker')
    expect(result.consumer?.num_pending).toBe(3)
  })
})

describe('consumer priority groups', () => {
  it('sends the priority settings on create', async () => {
    await createConsumer('conn-1', 'ORDERS', {
      name: 'worker',
      priority_policy: 'pinned_client',
      priority_groups: ['jobs'],
      priority_timeout: 60_000_000_000,
    })

    const req = createConsumerCall.mock.calls[0][0]
    expect(req.priorityPolicy).toBe(1)
    expect(req.priorityGroups).toEqual(['jobs'])
    expect(req.pinnedTtl).toEqual(create(DurationSchema, { seconds: 60n, nanos: 0 }))
  })

  it('defaults to no priority policy', async () => {
    await createConsumer('conn-1', 'ORDERS', { name: 'worker' })

    const req = createConsumerCall.mock.calls[0][0]
    expect(req.priorityPolicy).toBe(0)
    expect(req.priorityGroups).toEqual([])
    expect(req.pinnedTtl).toBeUndefined()
  })

  it('sends only the priority fields the update carries', async () => {
    await updateConsumer('conn-1', 'ORDERS', 'worker', { priority_policy: 'none' })

    const req = updateConsumerCall.mock.calls[0][0]
    expect(req.priorityPolicy).toBe(0)
    expect(req.priorityGroups).toEqual([])
    expect(req.pinnedTtl).toBeUndefined()
  })

  it('leaves priority untouched when the update omits it', async () => {
    await updateConsumer('conn-1', 'ORDERS', 'worker', { description: 'x' })

    expect(updateConsumerCall.mock.calls[0][0].priorityPolicy).toBeUndefined()
  })
})
