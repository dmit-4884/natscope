import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import * as api from '@/api/management'
import { toast } from '@/utils/toast'
import { CONNECTION_QUERY_PREFIX } from '@/hooks/useConnectionQuery'
import { useResetConsumer, useUnpinConsumer } from './useConsumerMutations'

vi.mock('@/api/management', () => ({
  resetConsumer: vi.fn(),
  unpinConsumer: vi.fn(),
}))

vi.mock('@/utils/toast', () => ({
  toast: { success: vi.fn(), error: vi.fn(), warning: vi.fn() },
}))

const resetConsumerMock = vi.mocked(api.resetConsumer)
const unpinConsumerMock = vi.mocked(api.unpinConsumer)

function setup() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  const invalidate = vi.spyOn(client, 'invalidateQueries')
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  )
  return { wrapper, invalidate }
}

describe('useResetConsumer', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('resets to a sequence and refreshes the stream and consumer list', async () => {
    resetConsumerMock.mockResolvedValue({ reset_seq: 42 })
    const { wrapper, invalidate } = setup()

    const { result } = renderHook(() => useResetConsumer('conn-1', 'ORDERS'), { wrapper })
    result.current.mutate({ name: 'worker', sequence: 42 })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(resetConsumerMock).toHaveBeenCalledWith('conn-1', 'ORDERS', 'worker', 42)
    expect(toast.success).toHaveBeenCalledWith('Consumer "worker" reset; delivery restarts at stream sequence 42')
    expect(invalidate).toHaveBeenCalledWith({ queryKey: [CONNECTION_QUERY_PREFIX, 'conn-1', 'stream', 'ORDERS'] })
    expect(invalidate).toHaveBeenCalledWith({ queryKey: [CONNECTION_QUERY_PREFIX, 'conn-1', 'consumers', 'ORDERS'] })
  })

  it('reports where delivery restarts after a reset that kept the ack floor', async () => {
    resetConsumerMock.mockResolvedValue({ reset_seq: 3 })
    const { wrapper } = setup()

    const { result } = renderHook(() => useResetConsumer('conn-1', 'ORDERS'), { wrapper })
    result.current.mutate({ name: 'worker' })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(resetConsumerMock).toHaveBeenCalledWith('conn-1', 'ORDERS', 'worker', undefined)
    expect(toast.success).toHaveBeenCalledWith('Consumer "worker" reset; delivery restarts at stream sequence 3')
  })

  it('toasts the server reason on failure', async () => {
    resetConsumerMock.mockRejectedValue(new Error('consumer reset requires NATS 2.14+ (connected server v2.12.3)'))
    const { wrapper } = setup()

    const { result } = renderHook(() => useResetConsumer('conn-1', 'ORDERS'), { wrapper })
    result.current.mutate({ name: 'worker' })

    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(toast.error).toHaveBeenCalledWith(
      'Failed to reset consumer: consumer reset requires NATS 2.14+ (connected server v2.12.3)',
    )
  })
})

describe('useUnpinConsumer', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('unpins the group and refreshes the consumer list', async () => {
    unpinConsumerMock.mockResolvedValue(undefined)
    const { wrapper, invalidate } = setup()

    const { result } = renderHook(() => useUnpinConsumer('conn-1', 'ORDERS'), { wrapper })
    result.current.mutate({ name: 'worker', group: 'jobs' })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(unpinConsumerMock).toHaveBeenCalledWith('conn-1', 'ORDERS', 'worker', 'jobs')
    expect(toast.success).toHaveBeenCalledWith('Unpinned the client of group "jobs"')
    expect(invalidate).toHaveBeenCalledWith({ queryKey: [CONNECTION_QUERY_PREFIX, 'conn-1', 'consumers', 'ORDERS'] })
  })

  it('toasts the server reason on failure', async () => {
    unpinConsumerMock.mockRejectedValue(new Error('consumer not found'))
    const { wrapper } = setup()

    const { result } = renderHook(() => useUnpinConsumer('conn-1', 'ORDERS'), { wrapper })
    result.current.mutate({ name: 'worker', group: 'jobs' })

    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(toast.error).toHaveBeenCalledWith('Failed to unpin: consumer not found')
  })
})
