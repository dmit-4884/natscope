import { describe, expect, it, vi } from 'vitest'
import type { ReactNode } from 'react'
import { renderHook } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { LiveStreamClient } from '../LiveStreamClient'
import { useProtoReloadInvalidation } from './useProtoReloadInvalidation'

const info = vi.hoisted(() => vi.fn())

vi.mock('@/utils/toast', () => ({ toast: { info } }))

describe('useProtoReloadInvalidation', () => {
  it('shows one toast when several live sessions report the same reload', () => {
    const queryClient = new QueryClient()
    const wrapper = ({ children }: { children: ReactNode }) => <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    const subscribe = {} as LiveStreamClient
    const stream = {} as LiveStreamClient
    renderHook(() => useProtoReloadInvalidation(subscribe), { wrapper })
    renderHook(() => useProtoReloadInvalidation(stream), { wrapper })

    subscribe.onProtoReload?.({ path: '', messages_count: 3 })
    stream.onProtoReload?.({ path: '', messages_count: 3 })

    expect(info).toHaveBeenCalledTimes(2)
    const [first, second] = info.mock.calls
    expect(first[1]?.id).toBeDefined()
    expect(second[1]?.id).toBe(first[1]?.id)
  })
})
