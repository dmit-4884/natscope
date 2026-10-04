import { describe, it, expect, beforeEach } from 'vitest'
import { act, renderHook } from '@testing-library/react'
import { clearAllSubscribeDrafts, useSubscribeDraft, withRecentSubjects } from './subscribeDraftStore'

describe('subscribeDraftStore', () => {
  beforeEach(() => clearAllSubscribeDrafts())

  it('keeps subjects per connection', () => {
    const a = renderHook(() => useSubscribeDraft('conn-a'))
    const b = renderHook(() => useSubscribeDraft('conn-b'))

    act(() => a.result.current[1]({ subjects: ['orders.>'] }))

    expect(a.result.current[0].subjects).toEqual(['orders.>'])
    expect(b.result.current[0].subjects).toEqual([])
  })
})

describe('withRecentSubjects', () => {
  it('puts the latest subjects first, without duplicates, capped', () => {
    const recent = ['a', 'b', 'c']
    expect(withRecentSubjects(recent, ['c', 'd'])).toEqual(['c', 'd', 'a', 'b'])
    expect(withRecentSubjects([], Array.from({ length: 20 }, (_, i) => `s${i}`))).toHaveLength(10)
  })
})
