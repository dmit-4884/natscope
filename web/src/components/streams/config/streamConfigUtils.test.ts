import { describe, it, expect } from 'vitest'
import type { StreamCreateRequest } from '@/types/management'
import type { StreamDetail } from '@/types/nats'
import { COMPRESSION_LABELS, canCreateStream, isMirrorConfigured, normalizeSubjects, streamToConfig } from './streamConfigUtils'

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
