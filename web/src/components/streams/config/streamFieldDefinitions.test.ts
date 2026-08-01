import { describe, it, expect } from 'vitest'
import type { StreamCreateRequest } from '@/types/management'
import {
  buildStreamCreatePayload,
  buildStreamUpdatePayload,
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
