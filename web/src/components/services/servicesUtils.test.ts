import { describe, it, expect } from 'vitest'
import type { MicroEndpoint, MicroService } from '@/api/discovery'
import {
  callDraftPatch,
  filterServices,
  formatErrorShare,
  formatRequestRate,
  formatWindowAverage,
  hasWildcard,
  serviceTotals,
  shortInstanceId,
} from './servicesUtils'

const endpoint = (over: Partial<MicroEndpoint> = {}): MicroEndpoint => ({
  name: 'Create',
  subject: 'orders.create',
  queue_group: 'q',
  metadata: {},
  ...over,
})

const service = (over: Partial<MicroService> = {}): MicroService => ({
  name: 'orders',
  description: 'Order API',
  versions: ['1.0.0'],
  instances: [],
  endpoints: [endpoint()],
  ...over,
})

describe('serviceTotals', () => {
  it('sums requests and errors and averages by request count', () => {
    const totals = serviceTotals(
      service({
        endpoints: [
          endpoint({ stats: { num_requests: 3, num_errors: 1, last_error: '', processing_time_ns: 30, average_processing_time_ns: 10 } }),
          endpoint({ name: 'Get', stats: { num_requests: 1, num_errors: 0, last_error: '', processing_time_ns: 10, average_processing_time_ns: 10 } }),
        ],
      }),
    )
    expect(totals).toEqual({ requests: 4, errors: 1, averageNs: 10 })
  })

  it('is null when no endpoint has stats', () => {
    expect(serviceTotals(service())).toBeNull()
  })
})

describe('filterServices', () => {
  const services = [service(), service({ name: 'billing', description: 'Invoices', endpoints: [endpoint({ subject: 'billing.pay' })] })]

  it('matches the name, description or an endpoint subject', () => {
    expect(filterServices(services, 'BILL').map((s) => s.name)).toEqual(['billing'])
    expect(filterServices(services, 'order api').map((s) => s.name)).toEqual(['orders'])
    expect(filterServices(services, 'billing.pay').map((s) => s.name)).toEqual(['billing'])
    expect(filterServices(services, '')).toBe(services)
  })
})

describe('callDraftPatch', () => {
  const draft = { subject: 'old.subject', payload: 'ping', replyTypes: { x: 'y' }, requestTypes: {} }

  it('targets the endpoint subject and keeps a plain JSON draft for an unmatched endpoint', () => {
    expect(callDraftPatch(endpoint(), draft)).toEqual({ subject: 'orders.create', payload: '' })
    expect(callDraftPatch(endpoint({ subject: 'old.subject' }), draft)).toEqual({ subject: 'old.subject' })
  })

  it('picks the request and reply types of a matched .proto method', () => {
    const patch = callDraftPatch(
      endpoint({
        proto_method: { source_id: 'src', service: 'shop.OrderService', method: 'Create', input_type: 'shop.In', output_type: 'shop.Out' },
      }),
      draft,
    )
    expect(patch).toEqual({
      subject: 'orders.create',
      payload: '{}',
      requestTypes: { 'orders.create': { messageType: 'shop.In', sourceId: 'src' } },
      replyTypes: { x: 'y', 'orders.create': 'src|shop.Out' },
    })
  })
})

describe('window formatting', () => {
  const window = (requests: number, errors: number, processingNs = 0) => ({ requests, errors, processingNs, seconds: 5 })

  it('formats the request rate', () => {
    expect(formatRequestRate(window(0, 0))).toBe('0 req/s')
    expect(formatRequestRate(window(2, 0))).toBe('0.4 req/s')
    expect(formatRequestRate(window(500, 0))).toBe('100 req/s')
  })

  it('formats the error share', () => {
    expect(formatErrorShare(window(0, 0))).toBe('—')
    expect(formatErrorShare(window(10, 0))).toBe('0%')
    expect(formatErrorShare(window(200, 3))).toBe('1.5%')
    expect(formatErrorShare(window(3, 1))).toBe('33%')
  })

  it('formats the average time and short instance ids', () => {
    expect(formatWindowAverage(window(0, 0))).toBe('—')
    expect(shortInstanceId('0MzWWnIEvjIRvBQ9tdosdr')).toBe('…tdosdr')
    expect(shortInstanceId('i-1')).toBe('i-1')
  })

  it('spots wildcard endpoint subjects', () => {
    expect(hasWildcard('calc.*')).toBe(true)
    expect(hasWildcard('calc.>')).toBe(true)
    expect(hasWildcard('calc.add')).toBe(false)
  })
})
