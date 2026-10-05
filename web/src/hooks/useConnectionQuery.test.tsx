import { describe, it, expect } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { CONNECTION_QUERY_PREFIX, useConnectionQuery } from './useConnectionQuery'

function setup() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: 30_000, refetchOnWindowFocus: false, gcTime: 60_000 } },
  })
  const wrapper = ({ children }: { children: ReactNode }) => <QueryClientProvider client={client}>{children}</QueryClientProvider>
  const observerOptions = () => client.getQueryCache().find({ queryKey: [CONNECTION_QUERY_PREFIX, 'conn-1', 'thing'] })!.observers[0].options
  return { wrapper, observerOptions }
}

describe('useConnectionQuery', () => {
  it('leaves the options a caller does not set to the client defaults', async () => {
    const { wrapper, observerOptions } = setup()
    const { result } = renderHook(() => useConnectionQuery({ key: ['thing'], connectionId: 'conn-1', fetcher: async () => 1 }), { wrapper })

    await waitFor(() => expect(result.current.data).toBe(1))
    expect(observerOptions()).toMatchObject({ retry: false, staleTime: 30_000, refetchOnWindowFocus: false, gcTime: 60_000 })
  })

  it('applies the options a caller sets', async () => {
    const { wrapper, observerOptions } = setup()
    const { result } = renderHook(
      () => useConnectionQuery({ key: ['thing'], connectionId: 'conn-1', fetcher: async () => 1, staleTime: 0, retry: 2 }),
      { wrapper },
    )

    await waitFor(() => expect(result.current.data).toBe(1))
    expect(observerOptions()).toMatchObject({ retry: 2, staleTime: 0 })
  })
})
