import { describe, it, expect } from 'vitest'
import type { StreamCreateRequest } from '@/types/management'
import { buildStreamCreatePayload } from './streamFieldDefinitions'

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
