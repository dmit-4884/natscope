import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { act, renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import * as statsApi from '@/api/stats'
import { AccessDeniedError } from '@/shared/domain/access'
import { CONSUMERS_REFRESH_MS, useConsumersOverview } from './useConsumersOverview'

vi.mock('@/api/stats', () => ({
  getConsumersOverview: vi.fn(),
}))

const getOverviewMock = vi.mocked(statsApi.getConsumersOverview)

function makeWrapper() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return ({ children }: { children: ReactNode }) => <QueryClientProvider client={client}>{children}</QueryClientProvider>
}

async function pollFor(times: number) {
  for (let i = 0; i < times; i++) {
    await act(async () => {
      await vi.advanceTimersByTimeAsync(CONSUMERS_REFRESH_MS)
    })
  }
}

describe('useConsumersOverview', () => {
  beforeEach(() => {
    getOverviewMock.mockReset()
    vi.useFakeTimers({ shouldAdvanceTime: true })
  })
  afterEach(() => {
    vi.useRealTimers()
  })

  it('polls while auto-refresh is on', async () => {
    getOverviewMock.mockResolvedValue({ consumers: [], streams: [], unreadable: [] })
    renderHook(() => useConsumersOverview('conn-1', { autoRefresh: true }), { wrapper: makeWrapper() })

    await waitFor(() => expect(getOverviewMock).toHaveBeenCalledTimes(1))
    await pollFor(2)
    expect(getOverviewMock.mock.calls.length).toBeGreaterThanOrEqual(3)
  })

  it('polls a slow listing less often, at a tenth of the time it takes', async () => {
    getOverviewMock.mockImplementation(
      () => new Promise((resolve) => setTimeout(() => resolve({ consumers: [], streams: [], unreadable: [] }), 2000)),
    )
    renderHook(() => useConsumersOverview('conn-1', { autoRefresh: true }), { wrapper: makeWrapper() })

    await waitFor(() => expect(getOverviewMock).toHaveBeenCalledTimes(1))
    await act(async () => {
      await vi.advanceTimersByTimeAsync(2000 + 15_000)
    })
    expect(getOverviewMock).toHaveBeenCalledTimes(1)
    await act(async () => {
      await vi.advanceTimersByTimeAsync(6000)
    })
    expect(getOverviewMock).toHaveBeenCalledTimes(2)
  })

  it('stops polling once the server refuses the listing', async () => {
    getOverviewMock.mockRejectedValue(
      new AccessDeniedError('refused', { status: 'denied', operation: 'publish', subject: '$JS.API.STREAM.LIST' }),
    )
    renderHook(() => useConsumersOverview('conn-1', { autoRefresh: true }), { wrapper: makeWrapper() })

    await waitFor(() => expect(getOverviewMock).toHaveBeenCalledTimes(1))
    await pollFor(3)
    expect(getOverviewMock).toHaveBeenCalledTimes(1)
  })
})
