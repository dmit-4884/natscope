import { describe, it, expect } from 'vitest'
import { coversSubject } from './subjectMatch'

describe('coversSubject', () => {
  it.each([
    ['>', 'orders.>'],
    ['orders.>', 'orders.>'],
    ['orders.>', 'orders.*.paid'],
    ['orders.*', 'orders.eu'],
    ['orders.*', 'orders.*'],
    ['orders.eu', 'orders.eu'],
  ])('%s takes every subject of %s', (filter, pattern) => {
    expect(coversSubject(filter, pattern)).toBe(true)
  })

  it.each([
    ['orders.*', 'orders.>'],
    ['orders.eu', 'orders.*'],
    ['orders.eu', 'orders.>'],
    ['orders.*.paid', 'orders.>'],
    ['orders.>', 'orders'],
    ['payments.>', 'orders.>'],
    ['orders.*', 'orders.eu.paid'],
  ])('%s misses some subjects of %s', (filter, pattern) => {
    expect(coversSubject(filter, pattern)).toBe(false)
  })
})
