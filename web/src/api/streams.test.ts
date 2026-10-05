import { describe, it, expect } from 'vitest'
import { create } from '@bufbuild/protobuf'
import { DurationSchema, TimestampSchema } from '@bufbuild/protobuf/wkt'
import { ConsumerInfoSchema, StreamInfoSchema } from '../gen/types/nats/nats_stream_pb'
import { readConsumerPauseState, toConsumerInfo, toStreamInfo } from './streams'

describe('toStreamInfo cluster replicas', () => {
  it('maps whether each replica is offline and how far it lags', () => {
    const info = toStreamInfo(
      create(StreamInfoSchema, {
        cluster: {
          name: 'hub',
          leader: 'hub-2',
          replicas: [
            { name: 'hub-1', current: true, active: create(DurationSchema, { nanos: 170_000_000 }) },
            { name: 'hub-3', offline: true, lag: 42n, active: create(DurationSchema, { seconds: 34_710n }) },
          ],
        },
      }),
    )

    expect(info.cluster?.replicas).toEqual([
      { name: 'hub-1', current: true, offline: false, active: 170_000_000, lag: 0 },
      { name: 'hub-3', current: false, offline: true, active: 34_710_000_000_000, lag: 42 },
    ])
  })
})

describe('readConsumerPauseState', () => {
  it('reports not paused without raw consumer info', () => {
    expect(readConsumerPauseState(undefined)).toEqual({ paused: false, pauseUntil: undefined })
  })

  it('reads paused plus the pause deadline from the server payload', () => {
    expect(
      readConsumerPauseState({
        paused: true,
        config: { pause_until: '2026-07-31T10:00:00Z' },
      }),
    ).toEqual({ paused: true, pauseUntil: '2026-07-31T10:00:00Z' })
  })

  it('treats a missing or non-boolean paused flag as not paused', () => {
    expect(readConsumerPauseState({ config: {} }).paused).toBe(false)
    expect(readConsumerPauseState({ paused: 'yes' }).paused).toBe(false)
  })

  it('ignores an empty or malformed pause_until', () => {
    expect(readConsumerPauseState({ paused: true, config: { pause_until: '' } }).pauseUntil)
      .toBeUndefined()
    expect(readConsumerPauseState({ paused: true, config: null }).pauseUntil).toBeUndefined()
    expect(readConsumerPauseState({ paused: true }).pauseUntil).toBeUndefined()
  })
})

describe('toConsumerInfo priority groups', () => {
  it('maps the priority settings and the pinned client', () => {
    const info = toConsumerInfo(
      create(ConsumerInfoSchema, {
        name: 'worker',
        config: {
          priorityPolicy: 3,
          priorityGroups: ['jobs'],
          pinnedTtl: create(DurationSchema, { seconds: 120n }),
        },
        priorityGroups: [
          { group: 'jobs', pinnedClientId: 'pin-1', pinnedTs: create(TimestampSchema, { seconds: 1_790_000_000n }) },
        ],
      }),
    )

    expect(info.config?.priority_policy).toBe('prioritized')
    expect(info.config?.priority_groups).toEqual(['jobs'])
    expect(info.config?.priority_timeout).toBe(120_000_000_000)
    expect(info.priority_groups).toEqual([{ group: 'jobs', pinned_client_id: 'pin-1', pinned_ts: 1_790_000_000_000 }])
  })

  it('omits priority when the consumer has none', () => {
    const info = toConsumerInfo(create(ConsumerInfoSchema, { name: 'plain', config: {} }))

    expect(info.config?.priority_policy).toBeUndefined()
    expect(info.config?.priority_groups).toBeUndefined()
    expect(info.priority_groups).toBeUndefined()
  })

  it('names the server-created flow control ack policy', () => {
    const info = toConsumerInfo(create(ConsumerInfoSchema, { name: 'sourcing', config: { ackPolicy: 3 } }))

    expect(info.config?.ack_policy).toBe('flow_control')
  })
})

describe('toConsumerInfo activity and delivery target', () => {
  it('maps when deliveries and acks last moved, and where a push consumer delivers', () => {
    const info = toConsumerInfo(
      create(ConsumerInfoSchema, {
        name: 'pusher',
        stream: 'ORDERS',
        config: { deliverSubject: 'deliver.orders' },
        delivered: { consumer: 5n, stream: 9n, lastActive: create(TimestampSchema, { seconds: 1_790_000_100n }) },
        ackFloor: { consumer: 4n, stream: 8n, lastActive: create(TimestampSchema, { seconds: 1_790_000_000n }) },
      }),
    )

    expect(info.delivered).toEqual({ consumer_seq: 5, stream_seq: 9, last_active: 1_790_000_100_000 })
    expect(info.ack_floor).toEqual({ consumer_seq: 4, stream_seq: 8, last_active: 1_790_000_000_000 })
    expect(info.config?.deliver_subject).toBe('deliver.orders')
  })

  it('leaves the activity unset before the first delivery', () => {
    const info = toConsumerInfo(create(ConsumerInfoSchema, { name: 'idle', config: {} }))

    expect(info.delivered?.last_active).toBeUndefined()
    expect(info.ack_floor?.last_active).toBeUndefined()
    expect(info.config?.deliver_subject).toBeUndefined()
  })
})
