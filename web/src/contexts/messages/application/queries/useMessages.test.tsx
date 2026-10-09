import { describe, expect, it, vi } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import * as api from '@/api/messages'
import type { MessagesResponse } from '@/api/messages'
import { useMessages } from './useMessages'

vi.mock('@/api/messages', () => ({ getMessages: vi.fn() }))

const page: MessagesResponse = { messages: [], has_more: false, next_seq: 0 }

function renderMessages() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const wrapper = ({ children }: { children: ReactNode }) => <QueryClientProvider client={client}>{children}</QueryClientProvider>
  vi.mocked(api.getMessages).mockResolvedValueOnce(page).mockReturnValue(new Promise(() => {}))
  return renderHook(
    ({ limit, subject }: { limit: number; subject?: string }) =>
      useMessages('ORDERS', { connection_id: 'conn-1', limit, subject_filter: subject }),
    { wrapper, initialProps: { limit: 50 } as { limit: number; subject?: string } },
  )
}

describe('useMessages', () => {
  it('keeps the page on screen while only its size changes', async () => {
    const { result, rerender } = renderMessages()
    await waitFor(() => expect(result.current.data).toBe(page))

    rerender({ limit: 100 })

    expect(result.current.data).toBe(page)
  })

  it('drops the page when the filter changes', async () => {
    const { result, rerender } = renderMessages()
    await waitFor(() => expect(result.current.data).toBe(page))

    rerender({ limit: 50, subject: 'orders.paid' })

    expect(result.current.data).toBeUndefined()
  })
})
