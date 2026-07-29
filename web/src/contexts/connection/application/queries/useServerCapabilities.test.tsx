import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import * as statsApi from '@/api/stats'
import type { ServerInfoResponse } from '@/api/stats'
import { useServerCapabilities } from './useServerCapabilities'

vi.mock('@/api/stats', () => ({
  getServerInfo: vi.fn(),
}))

const getServerInfoMock = vi.mocked(statsApi.getServerInfo)

function makeWrapper() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  )
}

function serverInfo(overrides: Partial<ServerInfoResponse>): ServerInfoResponse {
  return {
    server_id: 's1',
    server_name: 'nats-1',
    version: '2.14.2',
    host: 'localhost',
    port: 4222,
    cluster_name: '',
    max_payload: 1048576,
    connected_url: 'nats://localhost:4222',
    jetstream: true,
    auth_required: false,
    tls_required: false,
    connect_urls: [],
    ...overrides,
  }
}

describe('useServerCapabilities', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('fails open when the backend sends no capabilities', async () => {
    getServerInfoMock.mockResolvedValue(serverInfo({ capabilities: undefined }))

    const { result } = renderHook(() => useServerCapabilities('conn-1'), { wrapper: makeWrapper() })
    await waitFor(() => expect(getServerInfoMock).toHaveBeenCalled())

    expect(result.current.capabilities).toBeUndefined()
    expect(result.current.isSupported('consumerPause')).toBe(true)
    expect(result.current.isSupported('atomicPublish')).toBe(true)
    expect(result.current.unsupportedReason('consumerPause')).toBeUndefined()
  })

  it('gates only on explicit false and formats the reason', async () => {
    getServerInfoMock.mockResolvedValue(
      serverInfo({
        version: '2.10.24',
        capabilities: { api_level: 0, consumer_pause: false, message_ttl: false, atomic_publish: false },
      }),
    )

    const { result } = renderHook(() => useServerCapabilities('conn-1'), { wrapper: makeWrapper() })
    await waitFor(() => expect(result.current.capabilities).toBeDefined())

    expect(result.current.isSupported('consumerPause')).toBe(false)
    expect(result.current.unsupportedReason('consumerPause')).toBe(
      'Requires NATS 2.11+ (connected server is v2.10.24)',
    )
    expect(result.current.unsupportedReason('atomicPublish')).toBe(
      'Requires NATS 2.12+ (connected server is v2.10.24)',
    )
  })

  it('reports supported features with no reason', async () => {
    getServerInfoMock.mockResolvedValue(
      serverInfo({
        capabilities: { api_level: 4, consumer_pause: true, message_ttl: true, atomic_publish: true },
      }),
    )

    const { result } = renderHook(() => useServerCapabilities('conn-1'), { wrapper: makeWrapper() })
    await waitFor(() => expect(result.current.capabilities).toBeDefined())

    expect(result.current.isSupported('atomicPublish')).toBe(true)
    expect(result.current.unsupportedReason('atomicPublish')).toBeUndefined()
    expect(result.current.serverVersion).toBe('2.14.2')
  })
})
