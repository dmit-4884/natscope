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

const stored = vi.hoisted(() => ({ info: null as unknown }))

vi.mock('../activeConnectionStorage', () => ({ ...active, getStoredConnectionInfo: () => stored.info }))

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
    stored.info = null
    active.set(null)
    getConnectionsMock.mockResolvedValue([
      saved({ id: 'prod', readOnly: true, label: { text: 'PROD', color: 'red' } }),
      saved({ id: 'dev' }),
    ])
  })

  it('reports the active connection as read-only with its label', async () => {
    setActiveConnectionId('prod')
    const { result } = renderHook(() => useConnectionPolicy(), { wrapper: makeWrapper() })

    await waitFor(() => expect(result.current.label).toEqual({ text: 'PROD', color: 'red' }))
    expect(result.current.readOnly).toBe(true)
  })

  it('follows a switch to a writable connection', async () => {
    setActiveConnectionId('prod')
    const { result } = renderHook(() => useConnectionPolicy(), { wrapper: makeWrapper() })
    await waitFor(() => expect(result.current.label).not.toBeNull())

    act(() => setActiveConnectionId('dev'))

    expect(result.current.readOnly).toBe(false)
    expect(result.current.label).toBeNull()
  })

  it('stays read-only until the connection list says otherwise', async () => {
    let answer: (list: SavedConnection[]) => void = () => {}
    getConnectionsMock.mockReturnValue(new Promise((resolve) => (answer = resolve)))
    setActiveConnectionId('dev')
    const { result } = renderHook(() => useConnectionPolicy(), { wrapper: makeWrapper() })

    expect(result.current.readOnly).toBe(true)
    expect(result.current.known).toBe(false)

    act(() => answer([saved({ id: 'dev' })]))
    await waitFor(() => expect(result.current.readOnly).toBe(false))
    expect(result.current.known).toBe(true)
  })

  it('does not count a stored entry without a read-only flag as known', () => {
    getConnectionsMock.mockReturnValue(new Promise(() => {}))
    stored.info = { id: 'dev', name: 'dev', urls: ['nats://x'] }
    setActiveConnectionId('dev')
    const { result } = renderHook(() => useConnectionPolicy(), { wrapper: makeWrapper() })

    expect(result.current.known).toBe(false)
  })

  it('takes the stored policy of the active connection until the list answers', () => {
    getConnectionsMock.mockReturnValue(new Promise(() => {}))
    stored.info = { id: 'dev', name: 'dev', urls: ['nats://x'], readOnly: false, label: null }
    setActiveConnectionId('dev')
    const writable = renderHook(() => useConnectionPolicy(), { wrapper: makeWrapper() })
    expect(writable.result.current).toEqual({ readOnly: false, label: null, known: true })

    stored.info = { id: 'prod', name: 'prod', urls: ['nats://x'], readOnly: true, label: { text: 'PROD', color: 'red' } }
    setActiveConnectionId('prod')
    const prod = renderHook(() => useConnectionPolicy(), { wrapper: makeWrapper() })
    expect(prod.result.current).toEqual({ readOnly: true, label: { text: 'PROD', color: 'red' }, known: true })
  })
})
