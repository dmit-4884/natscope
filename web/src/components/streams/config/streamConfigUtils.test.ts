import { describe, it, expect } from 'vitest'
import type { StreamCreateRequest } from '@/types/management'
import type { ReplicaInfo, StreamDetail } from '@/types/nats'
import {
  COMPRESSION_LABELS,
  canCreateStream,
  createBlockedReason,
  validateStreamName,
  validateSubject,
  isMirrorConfigured,
  normalizeSubjects,
  replicaState,
  streamToConfig,
} from './streamConfigUtils'

describe('replicaState', () => {
  const replica = (over: Partial<ReplicaInfo> = {}): ReplicaInfo => ({
    name: 'hub-1',
    current: true,
    offline: false,
    active: 170_000_000,
    lag: 0,
    ...over,
  })

  it('reports a current replica as in sync', () => {
    expect(replicaState(replica())).toEqual({ variant: 'success', label: undefined, detail: 'In sync with the leader' })
  })

  it('reports how far a lagging replica is behind', () => {
    expect(replicaState(replica({ current: false, lag: 1234 }))).toEqual({
      variant: 'warning',
      label: '1.2K behind',
      detail: '1234 operations behind the leader',
    })
  })

  it('says a replica is catching up when the server reports no lag', () => {
    expect(replicaState(replica({ current: false }))).toEqual({
      variant: 'warning',
      label: 'catching up',
      detail: 'Catching up with the leader',
    })
  })

  it('reports an offline replica with the time it was last seen', () => {
    expect(replicaState(replica({ current: false, offline: true, active: 34_710_171_151_042 }))).toEqual({
      variant: 'error',
      label: 'offline',
      detail: 'Offline, last seen 9h 38m ago',
    })
  })

  it('reports an offline replica that was never seen', () => {
    expect(replicaState(replica({ current: false, offline: true, active: 0 }))).toEqual({
      variant: 'error',
      label: 'offline',
      detail: 'Offline',
    })
  })
})

function draft(over: Partial<StreamCreateRequest> = {}): StreamCreateRequest {
  return { name: 'ORDERS', subjects: [''], ...over }
}

describe('isMirrorConfigured', () => {
  it('is false without a mirror or with a blank mirror name', () => {
    expect(isMirrorConfigured({})).toBe(false)
    expect(isMirrorConfigured({ mirror: { name: '' } })).toBe(false)
    expect(isMirrorConfigured({ mirror: { name: '   ' } })).toBe(false)
  })

  it('is true once the mirror names a source stream', () => {
    expect(isMirrorConfigured({ mirror: { name: 'SOURCE' } })).toBe(true)
  })
})

describe('COMPRESSION_LABELS', () => {
  it('renders both algorithms with consistent casing', () => {
    expect(COMPRESSION_LABELS.none).toBe('None')
    expect(COMPRESSION_LABELS.s2).toBe('S2')
  })
})

describe('normalizeSubjects', () => {
  it('trims and drops blank rows', () => {
    expect(normalizeSubjects([' orders.> ', '', '   '])).toEqual(['orders.>'])
    expect(normalizeSubjects(undefined)).toEqual([])
  })
})

describe('canCreateStream', () => {
  it('requires a name', () => {
    expect(canCreateStream(draft({ name: '', subjects: ['orders.>'] }))).toBe(false)
    expect(canCreateStream(draft({ name: '  ', subjects: ['orders.>'] }))).toBe(false)
  })

  it('accepts a subject-only stream', () => {
    expect(canCreateStream(draft({ subjects: ['orders.>'] }))).toBe(true)
    expect(canCreateStream(draft({ subjects: [''] }))).toBe(false)
  })

  it('accepts a mirror stream without subjects', () => {
    expect(canCreateStream(draft({ subjects: [], mirror: { name: 'SOURCE' } }))).toBe(true)
    expect(canCreateStream(draft({ subjects: undefined, mirror: { name: 'SOURCE' } }))).toBe(true)
  })

  it('does not accept a mirror with a blank name and no subjects', () => {
    expect(canCreateStream(draft({ subjects: [''], mirror: { name: '' } }))).toBe(false)
  })
})

describe('streamToConfig', () => {
  function detail(config: Partial<StreamDetail['config']>): StreamDetail {
    return {
      name: 'FLAGS',
      subjects: ['flags.>'],
      messages: 0,
      bytes: 0,
      consumer_count: 0,
      created: 0,
      config: { retention: 'limits', max_msgs: -1, max_bytes: -1, max_age: 0, ...config },
      state: { messages: 0, bytes: 0, first_seq: 0, last_seq: 0, first_ts: 0, last_ts: 0 },
      consumers: [],
    }
  }

  it('carries the NATS 2.11–2.14 flags into the form value', () => {
    const value = streamToConfig(
      detail({
        allow_msg_counter: true,
        allow_msg_schedules: true,
        subject_delete_marker_ttl: 60_000_000_000,
        persist_mode: 'async',
        allow_batched: true,
      }),
    )
    expect(value.allow_msg_counter).toBe(true)
    expect(value.allow_msg_schedules).toBe(true)
    expect(value.subject_delete_marker_ttl).toBe(60_000_000_000)
    expect(value.persist_mode).toBe('async')
    expect(value.allow_batch_publish).toBe(true)
  })

  it('defaults persist mode to default', () => {
    expect(streamToConfig(detail({})).persist_mode).toBe('default')
  })
})

describe('validateStreamName', () => {
  it.each(['ORDERS', 'my-stream_v2', '_internal', 'orders2026'])('accepts %j', (name) => {
    expect(validateStreamName(name)).toBeUndefined()
  })

  it.each(['A3 bad name', 'a.b', 'a*b', 'a>b', 'a/b', 'a\\b', 'tab\tname'])('refuses %j and names the offending characters', (name) => {
    expect(validateStreamName(name)).toMatch(/spaces/)
  })

  it('says nothing about an empty name, the submit reason covers it', () => {
    expect(validateStreamName('')).toBeUndefined()
  })
})

describe('validateSubject', () => {
  it.each(['orders.>', 'orders.*.new', '>', 'a', '$KV.bucket.>'])('accepts %j', (subject) => {
    expect(validateSubject(subject)).toBeUndefined()
  })

  it.each([
    ['a..x', /empty/i],
    ['.a', /empty/i],
    ['a.', /empty/i],
    ['a b', /spaces/i],
    ['a.>.b', /last/i],
    ['a.b*', /whole/i],
  ])('refuses %j', (subject, message) => {
    expect(validateSubject(subject)).toMatch(message)
  })
})

describe('createBlockedReason', () => {
  it('states what is missing, in order', () => {
    expect(createBlockedReason(draft({ name: '', subjects: [''] }))).toBe('Enter a stream name.')
    expect(createBlockedReason(draft({ name: 'ORDERS', subjects: [''] }))).toBe('Add at least one subject, or configure a mirror.')
  })

  it('states the first broken rule', () => {
    expect(createBlockedReason(draft({ name: 'a b', subjects: ['x'] }))).toMatch(/spaces/)
    expect(createBlockedReason(draft({ name: 'ORDERS', subjects: ['a..x'] }))).toMatch(/a\.\.x.*empty/i)
  })

  it('is empty when the stream can be created', () => {
    expect(createBlockedReason(draft({ name: 'ORDERS', subjects: ['orders.>'] }))).toBeUndefined()
    expect(createBlockedReason(draft({ name: 'M', subjects: [], mirror: { name: 'SOURCE' } }))).toBeUndefined()
  })
})
