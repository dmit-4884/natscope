import { describe, it, expect, vi } from 'vitest'
import { renderHook } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { InvalidateQueryFilters } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { CONNECTION_QUERY_PREFIX } from '@/hooks/useConnectionQuery'
import { useHistoryRefreshOnModeChange } from './useHistoryRefreshOnModeChange'
import type { ViewMode } from './messageListUtils'

const CONNECTION_ID = 'conn-1'
const STREAM = 'ORDERS'

function setup() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const invalidate = vi.spyOn(client, 'invalidateQueries').mockResolvedValue(undefined)
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  )
  return { invalidate, wrapper }
}

function renderMode(
  wrapper: ({ children }: { children: ReactNode }) => ReactNode,
  initialMode: ViewMode,
  connectionId: string | null = CONNECTION_ID,
) {
  return renderHook(
    ({ mode }: { mode: ViewMode }) =>
      useHistoryRefreshOnModeChange(mode, connectionId, STREAM),
    { wrapper, initialProps: { mode: initialMode } },
  )
}

describe('useHistoryRefreshOnModeChange', () => {
  it('does not invalidate on the initial render', () => {
    const { invalidate, wrapper } = setup()

    renderMode(wrapper, 'history')

    expect(invalidate).not.toHaveBeenCalled()
  })

  it('invalidates messages and stream detail when leaving realtime', () => {
    const { invalidate, wrapper } = setup()

    const { rerender } = renderMode(wrapper, 'realtime')
    expect(invalidate).not.toHaveBeenCalled()

    rerender({ mode: 'history' })

    const keys = invalidate.mock.calls.map((call) => (call[0] as InvalidateQueryFilters).queryKey)
    expect(keys).toEqual([
      [CONNECTION_QUERY_PREFIX, CONNECTION_ID, 'messages', STREAM],
      [CONNECTION_QUERY_PREFIX, CONNECTION_ID, 'stream', STREAM],
    ])
  })

  it('stays quiet when entering realtime', () => {
    const { invalidate, wrapper } = setup()

    const { rerender } = renderMode(wrapper, 'history')
    rerender({ mode: 'realtime' })

    expect(invalidate).not.toHaveBeenCalled()
  })

  it('skips invalidation without an active connection', () => {
    const { invalidate, wrapper } = setup()

    const { rerender } = renderMode(wrapper, 'realtime', null)
    rerender({ mode: 'history' })

    expect(invalidate).not.toHaveBeenCalled()
  })
})
