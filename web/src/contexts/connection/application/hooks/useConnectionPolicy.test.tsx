import { describe, it, expect, vi, beforeEach } from 'vitest'
import { act, renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import * as connectionsApi from '@/api/connections'
import type { SavedConnection } from '@/api/connections'
import { useConnectionPolicy } from './useConnectionPolicy'

vi.mock('@/api/connections', () => ({ getConnections: vi.fn() }))

const active = vi.hoisted(() => {
  let id: string | null = null
  const listeners = new Set<() => void>()
  return {
    getActiveConnectionId: () => id,
    subscribeActiveConnection: (fn: () => void) => {
      listeners.add(fn)
      return () => listeners.delete(fn)
    },
    set(next: string | null) {
      id = next
      listeners.forEach((fn) => fn())
    },
  }
})

vi.mock('../activeConnectionStorage', () => active)

const setActiveConnectionId = (id: string) => active.set(id)

const getConnectionsMock = vi.mocked(connectionsApi.getConnections)

function makeWrapper() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return ({ children }: { children: ReactNode }) => <QueryClientProvider client={client}>{children}</QueryClientProvider>
}

const saved = (over: Partial<SavedConnection>): SavedConnection => ({
  id: 'x',
  name: 'x',
  urls: ['nats://x'],
  readOnly: false,
  createdAt: 0,
  updatedAt: 0,
  ...over,
})

describe('useConnectionPolicy', () => {
  beforeEach(() => {
    active.set(null)
    getConnectionsMock.mockResolvedValue([
      saved({ id: 'prod', readOnly: true, label: { text: 'PROD', color: 'red' } }),
      saved({ id: 'dev' }),
    ])
  })

  it('reports the active connection as read-only with its label', async () => {
    setActiveConnectionId('prod')
    const { result } = renderHook(() => useConnectionPolicy(), { wrapper: makeWrapper() })

    await waitFor(() => expect(result.current.readOnly).toBe(true))
    expect(result.current.label).toEqual({ text: 'PROD', color: 'red' })
  })

  it('follows a switch to a writable connection', async () => {
    setActiveConnectionId('prod')
    const { result } = renderHook(() => useConnectionPolicy(), { wrapper: makeWrapper() })
    await waitFor(() => expect(result.current.readOnly).toBe(true))

    act(() => setActiveConnectionId('dev'))

    expect(result.current.readOnly).toBe(false)
    expect(result.current.label).toBeNull()
  })
})
