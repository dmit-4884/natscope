import { describe, it, expect } from 'vitest'
import { arrangeSection, dropIndex, moveItem, togglePin } from './sectionArrangement'

const names = ['orders', 'Audit', 'events', 'billing', 'zeta']

describe('arrangeSection', () => {
  it('sorts by name when nothing is arranged', () => {
    expect(arrangeSection(names, { pinned: [], order: [] })).toEqual({
      pinned: [],
      rest: ['Audit', 'billing', 'events', 'orders', 'zeta'],
    })
  })

  it('puts pinned names first, then the manual order, then the rest by name', () => {
    expect(arrangeSection(names, { pinned: ['zeta', 'orders'], order: ['events', 'Audit'] })).toEqual({
      pinned: ['zeta', 'orders'],
      rest: ['events', 'Audit', 'billing'],
    })
  })

  it('skips names that no longer exist and names listed twice', () => {
    expect(arrangeSection(names, { pinned: ['gone', 'orders'], order: ['orders', 'gone', 'billing'] })).toEqual({
      pinned: ['orders'],
      rest: ['billing', 'Audit', 'events', 'zeta'],
    })
  })

  it('filters both groups by a case-insensitive substring', () => {
    expect(arrangeSection(names, { pinned: ['orders'], order: [] }, ' E ')).toEqual({
      pinned: ['orders'],
      rest: ['events', 'zeta'],
    })
  })
})

describe('togglePin', () => {
  it('pins at the end and removes the name from the manual order', () => {
    expect(togglePin({ pinned: ['a'], order: ['b', 'c'] }, 'b')).toEqual({ pinned: ['a', 'b'], order: ['c'] })
  })

  it('unpins', () => {
    expect(togglePin({ pinned: ['a', 'b'], order: ['c'] }, 'a')).toEqual({ pinned: ['b'], order: ['c'] })
  })
})

describe('moveItem', () => {
  it('moves an item to the target index', () => {
    expect(moveItem(['a', 'b', 'c', 'd'], 0, 2)).toEqual(['b', 'c', 'a', 'd'])
    expect(moveItem(['a', 'b', 'c', 'd'], 3, 0)).toEqual(['d', 'a', 'b', 'c'])
  })
})

describe('dropIndex', () => {
  it('turns a drop before or after a row into the final index', () => {
    expect(dropIndex(0, 2, false)).toBe(1)
    expect(dropIndex(0, 2, true)).toBe(2)
    expect(dropIndex(3, 1, false)).toBe(1)
    expect(dropIndex(3, 1, true)).toBe(2)
    expect(dropIndex(1, 1, true)).toBe(1)
  })
})
