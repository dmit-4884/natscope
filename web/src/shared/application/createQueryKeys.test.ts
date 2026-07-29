import { describe, it, expect } from 'vitest'
import { createQueryKeys } from './createQueryKeys'

describe('createQueryKeys', () => {
  const keys = createQueryKeys('streams')

  it('produces root namespace key', () => {
    expect(keys.all).toEqual(['streams'])
  })

  it('produces list parent and specific list keys', () => {
    expect(keys.lists()).toEqual(['streams', 'list'])
    expect(keys.list({ type: 'kv' })).toEqual(['streams', 'list', { type: 'kv' }])
    expect(keys.list()).toEqual(['streams', 'list', undefined])
  })

  it('produces detail parent and specific detail keys', () => {
    expect(keys.details()).toEqual(['streams', 'detail'])
    expect(keys.detail('abc')).toEqual(['streams', 'detail', 'abc'])
  })

  it('custom appends arbitrary suffix', () => {
    expect(keys.custom('stats', 'id-42')).toEqual(['streams', 'stats', 'id-42'])
  })

  it('keeps each namespace isolated', () => {
    const a = createQueryKeys('a')
    const b = createQueryKeys('b')
    expect(a.all).not.toEqual(b.all)
  })
})
