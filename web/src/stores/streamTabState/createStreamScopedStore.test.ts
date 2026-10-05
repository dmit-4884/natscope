import { describe, it, expect } from 'vitest'
import { act, renderHook } from '@testing-library/react'
import { createStreamScopedStore } from './createStreamScopedStore'

const scope = { connectionUrl: 'nats://localhost:4222', streamName: 'ORDERS' }

describe('createStreamScopedStore', () => {
  it('keeps the entry setter stable across renders, so effects can depend on it', () => {
    const memory = { getItem: () => null, setItem: () => undefined, removeItem: () => undefined }
    const store = createStreamScopedStore<{ selected: string | null }>({ name: 'scoped-test', defaults: { selected: null }, storage: memory })
    const { result, rerender } = renderHook(() => store.useEntry(scope))
    const firstSetter = result.current[1]

    act(() => firstSetter({ selected: 'billing' }))
    rerender()

    expect(result.current[0].selected).toBe('billing')
    expect(result.current[1]).toBe(firstSetter)
  })
})
