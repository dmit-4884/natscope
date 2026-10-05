import { describe, it, expect, vi, beforeEach } from 'vitest'
import { Code, ConnectError } from '@connectrpc/connect'
import type { ReactNode } from 'react'
import { QueryClientProvider } from '@tanstack/react-query'
import { createTestQueryClient, renderHook, waitFor } from '@/test/utils'
import { getMessage } from '@/api/messages'
import type { SelectedMessage } from '@/types/messages'
import type { Message } from '@/types/nats'
import { linkedSequence, useLinkedMessage } from './useLinkedMessage'

vi.mock('@/api/messages', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/api/messages')>()),
  getMessage: vi.fn(),
}))

const mockedGet = vi.mocked(getMessage)

function message(sequence: number): Message {
  return { sequence, subject: 'orders.paid', timestamp: Date.UTC(2026, 9, 5), data_base64: 'e30=', data_size: 2, content_type: 'json' }
}

function wrapper({ children }: { children: ReactNode }) {
  return <QueryClientProvider client={createTestQueryClient()}>{children}</QueryClientProvider>
}

function linked(param: string | null, selected: SelectedMessage | null = null) {
  const onFound = vi.fn()
  const onMissing = vi.fn()
  renderHook(
    () => useLinkedMessage({ connectionId: 'conn-1', streamName: 'ORDERS', param, selected, onFound, onMissing }),
    { wrapper },
  )
  return { onFound, onMissing }
}

describe('linkedSequence', () => {
  it('reads the sequence of a stored message and ignores live ones', () => {
    expect(linkedSequence('history-22')).toBe(22)
    expect(linkedSequence('live-abc')).toBeNull()
    expect(linkedSequence('history-')).toBeNull()
    expect(linkedSequence(null)).toBeNull()
  })
})

describe('useLinkedMessage', () => {
  beforeEach(() => {
    mockedGet.mockReset()
  })

  it('opens the message a link names', async () => {
    mockedGet.mockResolvedValue(message(22))
    const { onFound } = linked('history-22')

    await waitFor(() => expect(onFound).toHaveBeenCalledWith(expect.objectContaining({ id: 'history-22', sequence: 22 })))
    expect(mockedGet).toHaveBeenCalledWith('conn-1', 'ORDERS', 22)
  })

  it('does not fetch the message that is already open', () => {
    const { onFound } = linked('history-22', { id: 'history-22', sequence: 22, subject: 'orders.paid', timestamp: 0, data_base64: '', data_size: 0 })

    expect(mockedGet).not.toHaveBeenCalled()
    expect(onFound).not.toHaveBeenCalled()
  })

  it('reports a message that left the stream', async () => {
    mockedGet.mockRejectedValue(new ConnectError('gone', Code.NotFound))
    const { onFound, onMissing } = linked('history-7')

    await waitFor(() => expect(onMissing).toHaveBeenCalledWith(7, expect.any(ConnectError)))
    expect(onFound).not.toHaveBeenCalled()
  })
})
