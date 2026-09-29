import { describe, it, expect } from 'vitest'
import { create } from '@bufbuild/protobuf'
import { DurationSchema, TimestampSchema } from '@bufbuild/protobuf/wkt'
import { ConsumerInfoSchema } from '../gen/types/nats/nats_stream_pb'
import { readConsumerPauseState, toConsumerInfo } from './streams'

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
