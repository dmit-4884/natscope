import { describe, it, expect, vi } from 'vitest'
import { act, renderHook } from '@testing-library/react'
import { MutationCache, QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { shouldToastMutationError } from '@/utils/mutationErrorPolicy'
import { useCreateConnection, useTestConnection, useUpdateConnection } from './useConnectionMutations'

vi.mock('@/api/connections', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/api/connections')>()),
  createConnection: vi.fn().mockRejectedValue(new Error('Connection name already in use')),
  updateConnection: vi.fn().mockRejectedValue(new Error('Connection name already in use')),
  testConnection: vi.fn().mockRejectedValue(new Error('backend unavailable')),
}))

function setup() {
  const toasted: string[] = []
  const client = new QueryClient({
    mutationCache: new MutationCache({
      onError: (error, _vars, _ctx, mutation) => {
        if (shouldToastMutationError(mutation)) toasted.push((error as Error).message)
      },
    }),
  })
  const wrapper = ({ children }: { children: ReactNode }) => <QueryClientProvider client={client}>{children}</QueryClientProvider>
  return { toasted, wrapper }
}

describe('connection save mutations', () => {
  it('toast their errors by default', async () => {
    const { toasted, wrapper } = setup()
    const { result } = renderHook(() => useCreateConnection(), { wrapper })

    await act(async () => {
      await result.current.mutateAsync({ name: 'local', urls: ['nats://x'] }).catch(() => undefined)
    })
    expect(toasted).toEqual(['Connection name already in use'])
  })

  it('leave the error to a form that shows it itself', async () => {
    const { toasted, wrapper } = setup()
    const create = renderHook(() => useCreateConnection({ silent: true }), { wrapper })
    const update = renderHook(() => useUpdateConnection({ silent: true }), { wrapper })

    const test = renderHook(() => useTestConnection({ silent: true }), { wrapper })

    await act(async () => {
      await create.result.current.mutateAsync({ name: 'local', urls: ['nats://x'] }).catch(() => undefined)
      await update.result.current.mutateAsync({ id: 'c1', connection: { name: 'local' } }).catch(() => undefined)
      await test.result.current.mutateAsync({ urls: ['nats://x'] }).catch(() => undefined)
    })
    expect(toasted).toEqual([])
  })
})
