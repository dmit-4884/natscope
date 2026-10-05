import { describe, expect, it } from 'vitest'
import type { MicroEndpoint, MicroService } from '@/api/discovery'
import { healthOf, sampleOf, sumWindows, windowsBetween } from './serviceRates'

function endpoint(name: string, requests: number, errors: number, processingNs: number): MicroEndpoint {
  return {
    name,
    subject: `calc.${name}`,
    queue_group: 'q',
    metadata: {},
    stats: { num_requests: requests, num_errors: errors, last_error: '', processing_time_ns: processingNs, average_processing_time_ns: 0 },
  }
}

function service(instances: Record<string, MicroEndpoint[]>): MicroService {
  return {
    name: 'calc',
    description: '',
    versions: [],
    instances: Object.entries(instances).map(([id, endpoints]) => ({ id, version: '1.0.0', metadata: {}, endpoints })),
    endpoints: [],
  }
}

describe('windowsBetween', () => {
  it('turns two polls into requests, errors and time per window', () => {
    const before = sampleOf([service({ a: [endpoint('add', 10, 1, 1000)] })], 0)
    const after = sampleOf([service({ a: [endpoint('add', 30, 3, 5000)] })], 5000)

    const total = sumWindows(windowsBetween(before, after), () => true)
    expect(total).toEqual({ requests: 20, errors: 2, processingNs: 4000, seconds: 5 })
  })

  it('counts from zero after an instance restarts', () => {
    const before = sampleOf([service({ a: [endpoint('add', 100, 0, 0)] })], 0)
    const after = sampleOf([service({ a: [endpoint('add', 4, 1, 400)] })], 5000)

    expect(sumWindows(windowsBetween(before, after), () => true)).toMatchObject({ requests: 4, errors: 1 })
  })

  it('sums by instance or by endpoint', () => {
    const before = sampleOf([service({ a: [endpoint('add', 0, 0, 0)], b: [endpoint('add', 0, 0, 0)] })], 0)
    const after = sampleOf([service({ a: [endpoint('add', 5, 0, 0)], b: [endpoint('add', 7, 7, 0)] })], 1000)
    const windows = windowsBetween(before, after)

    expect(sumWindows(windows, (w) => w.instance === 'b')).toMatchObject({ requests: 7, errors: 7 })
    expect(sumWindows(windows, (w) => w.endpoint === 'add')).toMatchObject({ requests: 12, errors: 7 })
  })

  it('keeps measuring an endpoint the previous poll did not see, rather than take its lifetime totals for the window', () => {
    const before = sampleOf([service({ a: [{ ...endpoint('add', 0, 0, 0), stats: undefined }] })], 0)
    const after = sampleOf([service({ a: [endpoint('add', 1_000_000, 150_000, 0)] })], 5000)

    expect(sumWindows(windowsBetween(before, after), () => true)).toBeUndefined()
  })

  it('counts a new instance that started within the window from zero', () => {
    const before = sampleOf([service({ a: [endpoint('add', 10, 0, 0)] })], 0)
    const fresh = service({ a: [endpoint('add', 20, 0, 0)], b: [endpoint('add', 3, 1, 0)] })
    fresh.instances[1].started = 2000
    const after = sampleOf([fresh], 5000)

    expect(sumWindows(windowsBetween(before, after), (w) => w.instance === 'b')).toMatchObject({ requests: 3, errors: 1 })
  })

  it('has nothing to compare for endpoints without statistics', () => {
    const noStats = service({ a: [{ ...endpoint('add', 0, 0, 0), stats: undefined }] })
    expect(windowsBetween(sampleOf([noStats], 0), sampleOf([noStats], 5000))).toEqual([])
  })
})

describe('healthOf', () => {
  it('is unknown without a window and idle without requests', () => {
    expect(healthOf(undefined)).toBe('unknown')
    expect(healthOf({ requests: 0, errors: 0, processingNs: 0, seconds: 5 })).toBe('idle')
  })

  it('grades the error share of the window', () => {
    expect(healthOf({ requests: 1000, errors: 5, processingNs: 0, seconds: 5 })).toBe('ok')
    expect(healthOf({ requests: 100, errors: 5, processingNs: 0, seconds: 5 })).toBe('degraded')
    expect(healthOf({ requests: 10, errors: 3, processingNs: 0, seconds: 5 })).toBe('failing')
  })
})
