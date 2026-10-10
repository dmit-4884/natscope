import { beforeEach, describe, expect, it, vi } from 'vitest'
import { Code, ConnectError } from '@connectrpc/connect'
import { renderHook } from '@testing-library/react'

const captured = vi.hoisted(() => ({ options: undefined as undefined | { refetchInterval?: unknown } }))

vi.mock('@/hooks/useConnectionQuery', () => ({
  CONNECTION_QUERY_PREFIX: 'conn',
  useConnectionQuery: (opts: { refetchInterval?: unknown }) => {
    captured.options = opts
    return {}
  },
}))

import { useStreamDetail } from './useStreamList'

type Interval = (data: unknown, error: unknown) => number | false

describe('useStreamDetail', () => {
  beforeEach(() => {
    captured.options = undefined
  })

  it('refreshes periodically so the stream header does not freeze', () => {
    renderHook(() => useStreamDetail('ORDERS', 'c1'))
    const interval = captured.options?.refetchInterval as Interval

    expect(interval({}, null)).toBe(5000)
  })

  it('stops polling once the stream is gone', () => {
    renderHook(() => useStreamDetail('ORDERS', 'c1'))
    const interval = captured.options?.refetchInterval as Interval

    expect(interval({}, new ConnectError('stream not found', Code.NotFound))).toBe(false)
  })
})
