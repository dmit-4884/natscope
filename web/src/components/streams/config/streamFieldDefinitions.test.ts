import { describe, it, expect } from 'vitest'
import type { StreamCreateRequest } from '@/types/management'
import {
  STREAM_FIELDS,
  buildStreamCreatePayload,
  buildStreamUpdatePayload,
  enableOnlyLockReason,
  getIgnoredUpdateKeys,
  getStreamUpdateLockedKeys,
} from './streamFieldDefinitions'

function draft(over: Partial<StreamCreateRequest> = {}): StreamCreateRequest {
  return { name: 'ORDERS', subjects: ['orders.>'], retention: 'limits', storage: 'file', ...over }
}

describe('buildStreamCreatePayload', () => {
  it('keeps trimmed subjects and drops blank rows', () => {
    const payload = buildStreamCreatePayload(draft({ subjects: [' orders.> ', ''] }))
    expect(payload.subjects).toEqual(['orders.>'])
  })

  it('omits subjects entirely for a mirror stream', () => {
    const payload = buildStreamCreatePayload(draft({ subjects: ['orders.>'], mirror: { name: 'SOURCE' } }))
    expect('subjects' in payload).toBe(false)
    expect(payload.mirror).toEqual({ name: 'SOURCE' })
  })

  it('omits subjects when every row is blank', () => {
    const payload = buildStreamCreatePayload(draft({ subjects: ['', '  '] }))
    expect('subjects' in payload).toBe(false)
  })

  it('passes nested create-only sections through', () => {
    const payload = buildStreamCreatePayload(
      draft({
        placement: { cluster: 'east' },
        republish: { src: 'a.>', dest: 'b.>' },
        consumer_limits: { max_ack_pending: 10 },
      }),
    )
    expect(payload.placement).toEqual({ cluster: 'east' })
    expect(payload.republish).toEqual({ src: 'a.>', dest: 'b.>' })
    expect(payload.consumer_limits).toEqual({ max_ack_pending: 10 })
  })
})

describe('getStreamUpdateLockedKeys', () => {
  it('locks immutable form fields and the nested create-only sections', () => {
    const locked = getStreamUpdateLockedKeys()
    for (const key of ['name', 'retention', 'storage', 'num_replicas', 'mirror', 'placement', 'first_seq', 'no_ack']) {
      expect(locked).toContain(key)
    }
    expect(locked).not.toContain('subjects')
    expect(locked).not.toContain('metadata')
    expect(new Set(locked).size).toBe(locked.length)
  })
})

describe('buildStreamUpdatePayload', () => {
  it('drops immutable fields and keeps the mutable ones', () => {
    const payload = buildStreamUpdatePayload(
      draft({ retention: 'workqueue', first_seq: 9, mirror: { name: 'SOURCE' }, metadata: { team: 'core' } }),
    )
    expect(payload).not.toHaveProperty('name')
    expect(payload).not.toHaveProperty('retention')
    expect(payload).not.toHaveProperty('first_seq')
    expect(payload).not.toHaveProperty('mirror')
    expect(payload.subjects).toEqual(['orders.>'])
    expect(payload.metadata).toEqual({ team: 'core' })
  })
})

describe('getIgnoredUpdateKeys', () => {
  const original = draft({ subjects: ['orders.>'], metadata: { team: 'core' } })

  it('reports nothing when only mutable fields change', () => {
    expect(getIgnoredUpdateKeys(original, draft({ subjects: ['orders.eu'], metadata: { team: 'core' } }))).toEqual([])
    expect(getIgnoredUpdateKeys(original, { ...original, metadata: { team: 'platform' } })).toEqual([])
  })

  it('reports nothing when nothing changed', () => {
    expect(getIgnoredUpdateKeys(original, { ...original })).toEqual([])
  })

  it('reports immutable fields edited in the JSON view', () => {
    expect(getIgnoredUpdateKeys(original, { ...original, retention: 'workqueue' })).toEqual(['retention'])
    expect(getIgnoredUpdateKeys(original, { ...original, first_seq: 42, no_ack: true })).toEqual(['first_seq', 'no_ack'])
  })

  it('reports nested immutable sections added by hand', () => {
    expect(getIgnoredUpdateKeys(original, { ...original, mirror: { name: 'SOURCE' } })).toEqual(['mirror'])
    expect(getIgnoredUpdateKeys(original, { ...original, placement: { cluster: 'east' } })).toEqual(['placement'])
  })

  it('ignores key reordering and undefined-vs-absent', () => {
    const reordered: StreamCreateRequest = { storage: 'file', retention: 'limits', subjects: ['orders.>'], name: 'ORDERS', metadata: { team: 'core' } }
    expect(getIgnoredUpdateKeys(original, reordered)).toEqual([])
    expect(getIgnoredUpdateKeys(original, { ...original, mirror: undefined })).toEqual([])
  })
})

describe('NATS 2.11–2.14 stream flags', () => {
  const byKey = (key: string) => STREAM_FIELDS.find((f) => f.key === key)

  it.each([
    ['allow_msg_counter', 'msgCounters', false],
    ['allow_msg_schedules', 'msgSchedules', true],
    ['subject_delete_marker_ttl', 'messageTtl', true],
    ['persist_mode', 'asyncPersist', false],
    ['allow_batch_publish', 'batchPublish', true],
  ] as const)('%s is gated by %s and editable on update: %s', (key, capability, editableOnUpdate) => {
    const def = byKey(key)
    expect(def?.requiresCapability).toBe(capability)
    expect(def?.editableOnUpdate).toBe(editableOnUpdate)
    expect(def?.editableOnCreate).toBe(true)
  })

  it('keeps create-only flags out of the update payload', () => {
    const payload = buildStreamUpdatePayload(
      draft({
        allow_msg_counter: true,
        persist_mode: 'async',
        allow_msg_schedules: true,
        subject_delete_marker_ttl: 60_000_000_000,
        allow_batch_publish: true,
      }),
    )
    expect(payload).not.toHaveProperty('allow_msg_counter')
    expect(payload).not.toHaveProperty('persist_mode')
    expect(payload.allow_msg_schedules).toBe(true)
    expect(payload.subject_delete_marker_ttl).toBe(60_000_000_000)
    expect(payload.allow_batch_publish).toBe(true)
  })

  it('locks create-only flags in the JSON view', () => {
    const locked = getStreamUpdateLockedKeys()
    expect(locked).toContain('allow_msg_counter')
    expect(locked).toContain('persist_mode')
    expect(locked).not.toContain('allow_msg_schedules')
  })
})

describe('enableOnlyLockReason', () => {
  const ttl = STREAM_FIELDS.find((f) => f.key === 'allow_msg_ttl')!
  const schedules = STREAM_FIELDS.find((f) => f.key === 'allow_msg_schedules')!
  const direct = STREAM_FIELDS.find((f) => f.key === 'allow_direct')!

  it('locks an enable-only flag that the stream already has on', () => {
    expect(enableOnlyLockReason(ttl, 'edit', draft({ allow_msg_ttl: true }))).toBe('Cannot be disabled once enabled.')
    expect(enableOnlyLockReason(schedules, 'edit', draft({ allow_msg_schedules: true }))).toBe(
      'Cannot be disabled once enabled.',
    )
  })

  it('leaves the flag editable while it is still off', () => {
    expect(enableOnlyLockReason(ttl, 'edit', draft({ allow_msg_ttl: false }))).toBeUndefined()
    expect(enableOnlyLockReason(schedules, 'edit', draft())).toBeUndefined()
  })

  it('never locks in create mode or without the original config', () => {
    expect(enableOnlyLockReason(ttl, 'create', draft({ allow_msg_ttl: true }))).toBeUndefined()
    expect(enableOnlyLockReason(ttl, 'edit', null)).toBeUndefined()
  })

  it('ignores fields that are not enable-only', () => {
    expect(enableOnlyLockReason(direct, 'edit', draft({ allow_direct: true }))).toBeUndefined()
  })
})

describe('duration field labels', () => {
  it('names nanoseconds on every duration field, like Max Age', () => {
    for (const def of STREAM_FIELDS.filter((f) => f.type === 'duration_ns')) {
      expect(def.label).toMatch(/\(ns\)$/)
    }
  })

  it('does not tell the user to type seconds into a nanosecond field', () => {
    const marker = STREAM_FIELDS.find((f) => f.key === 'subject_delete_marker_ttl')!
    expect(marker.helperText).toContain('1000000000')
  })
})
