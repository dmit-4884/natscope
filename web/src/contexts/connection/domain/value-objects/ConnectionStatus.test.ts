import { describe, it, expect } from 'vitest'
import { resolveConnectionStatus } from './ConnectionStatus'

describe('resolveConnectionStatus', () => {
  it('reports connecting while the first health probe is in flight', () => {
    expect(resolveConnectionStatus({ isPending: true, hasError: false })).toBe('connecting')
  })

  it('never claims connected while an error is present', () => {
    expect(
      resolveConnectionStatus({ reported: 'connected', isPending: false, hasError: true }),
    ).toBe('disconnected')
  })

  it('prefers the error over a pending probe', () => {
    expect(resolveConnectionStatus({ isPending: true, hasError: true })).toBe('disconnected')
  })

  it('mirrors the server-reported status once healthy data arrives', () => {
    expect(
      resolveConnectionStatus({ reported: 'connected', isPending: false, hasError: false }),
    ).toBe('connected')
    expect(
      resolveConnectionStatus({ reported: 'reconnecting', isPending: false, hasError: false }),
    ).toBe('reconnecting')
    expect(
      resolveConnectionStatus({ reported: 'disconnected', isPending: false, hasError: false }),
    ).toBe('disconnected')
  })

  it('falls back to disconnected when the query is idle with no data', () => {
    expect(resolveConnectionStatus({ isPending: false, hasError: false })).toBe('disconnected')
  })
})
