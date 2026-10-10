import { describe, expect, it } from 'vitest'
import { overlappingStreams } from './overlappingStreams'

const streams = [
  { name: 'ORDERS', subjects: ['orders.>'] },
  { name: 'PAYMENTS', subjects: ['pay.*', 'refund.card'] },
  { name: 'MIRROR', subjects: [] },
  { name: 'OTHER', subjects: ['other.x'] },
]

describe('overlappingStreams', () => {
  it('names each stream that takes a subject the new one asks for', () => {
    expect(overlappingStreams(['orders.new', 'pay.card'], streams)).toEqual([
      { name: 'ORDERS', subjects: ['orders.>'] },
      { name: 'PAYMENTS', subjects: ['pay.*'] },
    ])
  })

  it('catches a wider pattern covering an existing narrower one', () => {
    expect(overlappingStreams(['>'], streams).map((s) => s.name)).toEqual(['ORDERS', 'PAYMENTS', 'OTHER'])
  })

  it('ignores the stream being edited and blank subjects', () => {
    expect(overlappingStreams(['orders.>', ''], streams, 'ORDERS')).toEqual([])
  })

  it('finds nothing for a free subject', () => {
    expect(overlappingStreams(['free.subject'], streams)).toEqual([])
  })
})
