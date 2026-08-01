import { describe, it, expect } from 'vitest'
import type { StreamCreateRequest } from '@/types/management'
import { canCreateStream, formatCompression, isMirrorConfigured, normalizeSubjects } from './streamConfigUtils'

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

describe('formatCompression', () => {
  it('renders both algorithms with consistent casing', () => {
    expect(formatCompression('none')).toBe('None')
    expect(formatCompression('s2')).toBe('S2')
    expect(formatCompression('S2')).toBe('S2')
  })

  it('falls back to the readable form for unknown values', () => {
    expect(formatCompression('zstd')).toBe('Zstd')
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
