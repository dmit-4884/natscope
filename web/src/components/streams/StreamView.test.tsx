import { beforeEach, describe, expect, it, vi } from 'vitest'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { Code, ConnectError } from '@connectrpc/connect'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'

const detail = vi.hoisted(() => ({ data: undefined as unknown, error: null as unknown }))

vi.mock('@/contexts/streams', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/contexts/streams')>()),
  useStreamDetail: () => detail,
  useStreamRelations: () => ({}),
  useConsumers: () => ({}),
}))
vi.mock('./useLinkedMessage', () => ({ useLinkedMessage: () => {} }))
vi.mock('../messages/unified/useMessageNavigation', () => ({
  useMessageNavigation: () => ({}),
}))
vi.mock('react-router-dom', async (importOriginal) => ({
  ...(await importOriginal<typeof import('react-router-dom')>()),
  useOutletContext: () => ({ connectionId: 'c1', currentConnection: { urls: ['nats://a'] }, handleOpenMappings: vi.fn() }),
}))

import StreamView from './StreamView'

function notFound() {
  return new ConnectError('stream not found', Code.NotFound)
}

function renderView(client = new QueryClient()) {
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={['/streams/ORDERS/messages']}>
        <Routes>
          <Route path="/streams/:streamName/*" element={<StreamView />} />
          <Route path="/streams" element={<p>streams list</p>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
  return client
}

describe('StreamView', () => {
  beforeEach(() => {
    detail.data = undefined
    detail.error = null
  })

  it('says the stream was deleted when it vanishes after loading, and refreshes the stream list', async () => {
    detail.data = { name: 'ORDERS', subjects: ['orders.>'] }
    detail.error = notFound()
    const client = new QueryClient()
    const invalidate = vi.spyOn(client, 'invalidateQueries')

    renderView(client)

    expect(screen.getByText('This stream was deleted')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Back to streams' })).toBeInTheDocument()
    await waitFor(() => expect(invalidate).toHaveBeenCalledWith({ queryKey: ['conn', 'c1', 'streams'] }))
  })

  it('keeps the plain not-found wording for a stream that never loaded', () => {
    detail.error = notFound()

    renderView()

    expect(screen.getByText('Stream "ORDERS" not found')).toBeInTheDocument()
  })
})
