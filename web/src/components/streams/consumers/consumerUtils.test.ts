import { describe, it, expect } from 'vitest'
import type { ConsumerInfo } from '@/types/nats'
import type { ConsumerCreateRequest } from '@/types/management'
import { consumerToConfig, toConsumerUpdateRequest } from './consumerUtils'

const base: ConsumerCreateRequest = {
  name: 'orders-worker',
  ack_wait: 30_000_000_000,
  max_ack_pending: 1000,
}

describe('consumerToConfig', () => {
  it('marks a consumer without a durable name as ephemeral', () => {
    const consumer = { name: 'auto-1', config: {} } as ConsumerInfo
    expect(consumerToConfig(consumer).ephemeral).toBe(true)
  })

  it('marks a consumer with a durable name as durable', () => {
    const consumer = { name: 'worker', config: { durable_name: 'worker' } } as ConsumerInfo
    expect(consumerToConfig(consumer).ephemeral).toBe(false)
  })

  it('carries the multi-subject filter list into the form draft', () => {
    const consumer = {
      name: 'worker',
      config: { filter_subjects: ['orders.created', 'orders.paid'] },
    } as ConsumerInfo
    expect(consumerToConfig(consumer).filter_subjects).toEqual(['orders.created', 'orders.paid'])
  })
})

describe('toConsumerUpdateRequest', () => {
  it('omits both filter fields when nothing changed', () => {
    const original = { ...base, filter_subject: 'orders.>' }
    const request = toConsumerUpdateRequest(original, { ...original })
    expect('filter_subject' in request).toBe(false)
    expect('filter_subjects' in request).toBe(false)
  })

  it('sends the new single filter when it changed', () => {
    const original = { ...base, filter_subject: 'orders.>' }
    const request = toConsumerUpdateRequest(original, { ...original, filter_subject: 'orders.paid' })
    expect(request.filter_subject).toBe('orders.paid')
    expect('filter_subjects' in request).toBe(false)
  })

  it('sends an empty single filter when the user cleared it', () => {
    const original = { ...base, filter_subject: 'orders.>' }
    const request = toConsumerUpdateRequest(original, { ...original, filter_subject: '' })
    expect(request.filter_subject).toBe('')
  })

  it('sends the list when the user replaced a single filter with several', () => {
    const original = { ...base, filter_subject: 'orders.>' }
    const request = toConsumerUpdateRequest(original, {
      ...original,
      filter_subject: '',
      filter_subjects: ['orders.created', 'orders.paid'],
    })
    expect(request.filter_subjects).toEqual(['orders.created', 'orders.paid'])
    expect('filter_subject' in request).toBe(false)
  })

  it('never sends both filter fields at once', () => {
    const original = { ...base, filter_subject: 'orders.>' }
    const request = toConsumerUpdateRequest(original, {
      ...original,
      filter_subject: 'orders.paid',
      filter_subjects: ['orders.created'],
    })
    expect(request.filter_subjects).toEqual(['orders.created'])
    expect('filter_subject' in request).toBe(false)
  })

  it('clears the filters via an empty single filter when the list was emptied', () => {
    const original = { ...base, filter_subjects: ['orders.created', 'orders.paid'] }
    const request = toConsumerUpdateRequest(original, { ...original, filter_subjects: [] })
    expect(request.filter_subject).toBe('')
    expect('filter_subjects' in request).toBe(false)
  })

  it('ignores blank rows left behind by the list editor', () => {
    const original = { ...base, filter_subjects: ['orders.created'] }
    const request = toConsumerUpdateRequest(original, {
      ...original,
      filter_subjects: ['orders.created', ''],
    })
    expect('filter_subject' in request).toBe(false)
    expect('filter_subjects' in request).toBe(false)
  })

  it('keeps forwarding the mutable non-filter fields', () => {
    const request = toConsumerUpdateRequest(base, { ...base, description: 'renamed', max_deliver: 5 })
    expect(request.description).toBe('renamed')
    expect(request.max_deliver).toBe(5)
  })
})
