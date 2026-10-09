import { describe, it, expect, vi, beforeEach } from 'vitest'
import { create } from '@bufbuild/protobuf'
import { DurationSchema } from '@bufbuild/protobuf/wkt'

import { KVBucketInfoSchema } from '../gen/types/nats/nats_kv_pb'
import { ConsumerInfoSchema, StreamInfoSchema } from '../gen/types/nats/nats_stream_pb'

const createConsumerCall = vi.fn()
const updateConsumerCall = vi.fn()
const putKVKeyCall = vi.fn()
const getKVKeyCall = vi.fn()
const createStreamCall = vi.fn()
const updateStreamCall = vi.fn()
const resetConsumerCall = vi.fn()
const createKVBucketCall = vi.fn()
const updateKVBucketCall = vi.fn()

vi.mock('./grpc/clients', () => ({
  streamsClient: {},
  managementClient: {
    createConsumer: createConsumerCall,
    updateConsumer: updateConsumerCall,
    putKVKey: putKVKeyCall,
    getKVKey: getKVKeyCall,
    createStream: createStreamCall,
    updateStream: updateStreamCall,
    resetConsumer: resetConsumerCall,
    createKVBucket: createKVBucketCall,
    updateKVBucket: updateKVBucketCall,
  },
}))

const { createConsumer, updateConsumer, putKVKey, getKVKey, createStream, updateStream, resetConsumer, createKVBucket, updateKVBucket } =
  await import('./management')

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

  it('sends a key TTL as a Duration', async () => {
    await putKVKey('conn-1', 'CONFIG', 'session', 'v', undefined, 90_000_000_000)

    expect(putKVKeyCall.mock.calls[0][0].ttl.seconds).toBe(90n)
  })

  it('sends no TTL for a plain write', async () => {
    await putKVKey('conn-1', 'CONFIG', 'session', 'v')

    expect(putKVKeyCall.mock.calls[0][0].ttl).toBeUndefined()
  })

  it('sends no expected revision when creating a key', async () => {
    await putKVKey('conn-1', 'config', 'greeting', 'hello')

    expect(putKVKeyCall.mock.calls[0][0].revision).toBe(0n)
  })

  it('sends text as base64 bytes', async () => {
    await putKVKey('conn-1', 'config', 'greeting', 'hi')

    expect(putKVKeyCall.mock.calls[0][0].payload).toEqual({ case: 'value', value: 'aGk=' })
  })

  it('sends JSON for the server to encode as Protobuf', async () => {
    await putKVKey('conn-1', 'config', 'limits', {
      messageType: 'shop.Limits',
      sourceId: 'src-1',
      json: '{"max":3}',
      framing: { kind: 'grpc', schemaId: 0, prefix: new Uint8Array(), suffix: new Uint8Array() },
      fingerprint: 'fp-1',
    })

    const payload = putKVKeyCall.mock.calls[0][0].payload
    expect(payload.case).toBe('proto')
    expect(payload.value).toMatchObject({
      messageType: 'shop.Limits',
      sourceId: 'src-1',
      json: '{"max":3}',
      fingerprint: 'fp-1',
    })
    expect(payload.value.framing.kind).toBe(1)
  })
})

describe('KV bucket settings', () => {
  const pbBucket = create(KVBucketInfoSchema, {
    bucket: 'CONFIG',
    maxValueSize: 1024,
    maxBytes: 4096n,
    isCompressed: true,
    limitMarkerTtl: create(DurationSchema, { seconds: 5n }),
  })

  beforeEach(() => {
    createKVBucketCall.mockResolvedValue({ bucket: pbBucket })
    updateKVBucketCall.mockResolvedValue({ bucket: pbBucket })
  })

  it('creates a bucket with compression and the key TTL marker', async () => {
    await createKVBucket('conn-1', { bucket: 'CONFIG', compression: true, limit_marker_ttl: 5_000_000_000 })

    const { config } = createKVBucketCall.mock.calls[0][0]
    expect(config.compression).toBe(true)
    expect(config.limitMarkerTtl.seconds).toBe(5n)
  })

  it('creates a bucket with the placement, mirror, sources and republish the form sets', async () => {
    await createKVBucket('conn-1', {
      bucket: 'CONFIG',
      placement: { cluster: 'east', tags: ['ssd'] },
      mirror: {
        name: 'ORIGIN',
        opt_start_seq: 5,
        opt_start_time: '2026-10-01T00:00:00Z',
        filter_subject: '$KV.ORIGIN.a.>',
        subject_transforms: [{ src: '$KV.ORIGIN.>', dest: '$KV.CONFIG.>' }],
        external: { api: '$JS.hub.API', deliver: 'deliver.hub' },
      },
      sources: [{ name: 'OTHER' }],
      republish: { src: '>', dest: 'repub.>', headers_only: true },
    })

    const { config } = createKVBucketCall.mock.calls[0][0]
    expect(config.placement).toEqual({ cluster: 'east', tags: ['ssd'] })
    expect(config.mirror).toMatchObject({
      name: 'ORIGIN',
      optStartSeq: 5n,
      filterSubject: '$KV.ORIGIN.a.>',
      subjectTransforms: [{ source: '$KV.ORIGIN.>', destination: '$KV.CONFIG.>' }],
      external: { apiPrefix: '$JS.hub.API', deliverPrefix: 'deliver.hub' },
    })
    expect(new Date(Number(config.mirror.optStartTime.seconds) * 1000).toISOString()).toBe('2026-10-01T00:00:00.000Z')
    expect(config.sources).toHaveLength(1)
    expect(config.sources[0].name).toBe('OTHER')
    expect(config.republish).toEqual({ src: '>', dest: 'repub.>', headersOnly: true })
  })

  it('maps limits and the key TTL marker back from the bucket', async () => {
    const info = await createKVBucket('conn-1', { bucket: 'CONFIG' })

    expect(info.max_value_size).toBe(1024)
    expect(info.max_bytes).toBe(4096)
    expect(info.is_compressed).toBe(true)
    expect(info.limit_marker_ttl).toBe(5_000_000_000)
  })

  it('sends every editable setting on update', async () => {
    await updateKVBucket('conn-1', 'CONFIG', {
      description: 'flags',
      history: 4,
      ttl: 60_000_000_000,
      max_value_size: 512,
      max_bytes: 2048,
      num_replicas: 3,
      compression: true,
      limit_marker_ttl: 2_000_000_000,
      metadata: { team: 'core' },
    })

    const req = updateKVBucketCall.mock.calls[0][0]
    expect(req.bucket).toBe('CONFIG')
    expect(req.settings).toMatchObject({
      description: 'flags',
      history: 4,
      maxValueSize: 512,
      maxBytes: 2048n,
      numReplicas: 3,
      compression: true,
      metadata: { team: 'core' },
    })
    expect(req.settings.ttl.seconds).toBe(60n)
    expect(req.settings.limitMarkerTtl.seconds).toBe(2n)
  })
})

describe('getKVKey', () => {
  it('maps a decoded Protobuf value', async () => {
    getKVKeyCall.mockResolvedValue({
      entry: {
        key: 'limits',
        value: 'CgNtYXg=',
        revision: 3n,
        operation: 'put',
        decoded: {
          data: '{"max":3}',
          messageType: 'shop.Limits',
          sourceId: 'src-1',
          auto: true,
          validBytes: 0,
          unknownFields: [{ number: 9 }],
        },
      },
    })

    const entry = await getKVKey('conn-1', 'config', 'limits')
    expect(entry.revision).toBe(3)
    expect(entry.decoded).toEqual({
      data: { max: 3 },
      messageType: 'shop.Limits',
      sourceId: 'src-1',
      auto: true,
      error: undefined,
      validBytes: 0,
      unknownFields: 1,
    })
  })

  it('leaves values without a decoded form alone', async () => {
    getKVKeyCall.mockResolvedValue({ entry: { key: 'plain', value: 'aGk=', revision: 1n, operation: 'put' } })

    expect((await getKVKey('conn-1', 'config', 'plain')).decoded).toBeUndefined()
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
