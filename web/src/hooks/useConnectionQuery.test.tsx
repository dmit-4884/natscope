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

  it('keeps the previous answer on screen while a related key loads', async () => {
    const { wrapper } = setup()
    const { result, rerender } = renderHook(
      ({ size }: { size: number }) =>
        useConnectionQuery({
          key: ['page', 'ORDERS', size],
          connectionId: 'conn-1',
          fetcher: () => (size === 50 ? Promise.resolve('first page') : new Promise<string>(() => {})),
          keepPreviousWhen: (previous) => previous[1] === 'ORDERS',
        }),
      { wrapper, initialProps: { size: 50 } },
    )
    await waitFor(() => expect(result.current.data).toBe('first page'))

    rerender({ size: 100 })

    expect(result.current.data).toBe('first page')
  })

  it('never carries an answer over to another connection', async () => {
    const { wrapper } = setup()
    const { result, rerender } = renderHook(
      ({ conn }: { conn: string }) =>
        useConnectionQuery({
          key: ['page', 'ORDERS'],
          connectionId: conn,
          fetcher: () => (conn === 'conn-1' ? Promise.resolve('conn-1 page') : new Promise<string>(() => {})),
          keepPreviousWhen: () => true,
        }),
      { wrapper, initialProps: { conn: 'conn-1' } },
    )
    await waitFor(() => expect(result.current.data).toBe('conn-1 page'))

    rerender({ conn: 'conn-2' })

    expect(result.current.data).toBeUndefined()
  })
})
