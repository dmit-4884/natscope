import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import * as statsApi from '@/api/stats'
import type { ServerInfoResponse } from '@/api/stats'
import { useServerCapabilities, type CapabilityKey } from './useServerCapabilities'

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

type Capabilities = NonNullable<ServerInfoResponse['capabilities']>

function capabilities(level: number): Capabilities {
  return {
    api_level: level,
    consumer_pause: level >= 1,
    message_ttl: level >= 1,
    priority_groups: level >= 1,
    atomic_publish: level >= 2,
    msg_counters: level >= 2,
    msg_schedules: level >= 2,
    priority_prioritized: level >= 2,
    async_persist: level >= 2,
    consumer_reset: level >= 4,
    cron_schedules: level >= 4,
    batch_publish: level >= 4,
  }
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
    expect(result.current.isSupported('consumerReset')).toBe(true)
    expect(result.current.unsupportedReason('consumerPause')).toBeUndefined()
  })

  it('gates only on explicit false and formats the reason', async () => {
    getServerInfoMock.mockResolvedValue(serverInfo({ version: '2.10.24', capabilities: capabilities(0) }))

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

  it.each<[CapabilityKey, string]>([
    ['priorityGroups', '2.11'],
    ['msgCounters', '2.12'],
    ['msgSchedules', '2.12'],
    ['priorityPrioritized', '2.12'],
    ['asyncPersist', '2.12'],
    ['consumerReset', '2.14'],
    ['cronSchedules', '2.14'],
    ['batchPublish', '2.14'],
  ])('labels %s with NATS %s+', async (key, since) => {
    getServerInfoMock.mockResolvedValue(serverInfo({ version: '2.10.24', capabilities: capabilities(0) }))

    const { result } = renderHook(() => useServerCapabilities('conn-1'), { wrapper: makeWrapper() })
    await waitFor(() => expect(result.current.capabilities).toBeDefined())

    expect(result.current.isSupported(key)).toBe(false)
    expect(result.current.unsupportedReason(key)).toBe(`Requires NATS ${since}+ (connected server is v2.10.24)`)
  })

  it('splits 2.12 from 2.14 features on a level 3 server', async () => {
    getServerInfoMock.mockResolvedValue(serverInfo({ version: '2.12.15', capabilities: capabilities(3) }))

    const { result } = renderHook(() => useServerCapabilities('conn-1'), { wrapper: makeWrapper() })
    await waitFor(() => expect(result.current.capabilities).toBeDefined())

    expect(result.current.isSupported('msgSchedules')).toBe(true)
    expect(result.current.isSupported('cronSchedules')).toBe(false)
    expect(result.current.isSupported('consumerReset')).toBe(false)
    expect(result.current.unsupportedReason('consumerReset')).toBe(
      'Requires NATS 2.14+ (connected server is v2.12.15)',
    )
  })

  it('reports supported features with no reason', async () => {
    getServerInfoMock.mockResolvedValue(serverInfo({ capabilities: capabilities(4) }))

    const { result } = renderHook(() => useServerCapabilities('conn-1'), { wrapper: makeWrapper() })
    await waitFor(() => expect(result.current.capabilities).toBeDefined())

    expect(result.current.isSupported('atomicPublish')).toBe(true)
    expect(result.current.isSupported('batchPublish')).toBe(true)
    expect(result.current.unsupportedReason('atomicPublish')).toBeUndefined()
    expect(result.current.serverVersion).toBe('2.14.2')
  })
})
