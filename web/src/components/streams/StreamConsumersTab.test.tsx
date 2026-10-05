import { describe, it, expect, vi } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import { render, screen } from '@/test/utils'
import StreamConsumersTab from './StreamConsumersTab'

const policy = vi.hoisted(() => ({ readOnly: false }))
const list = vi.hoisted(() => ({ isLoading: false }))

vi.mock('react-router-dom', async (importOriginal) => ({
  ...(await importOriginal<typeof import('react-router-dom')>()),
  useOutletContext: () => ({ scope: 'conn-1:ORDERS', connectionId: 'conn-1', streamName: 'ORDERS' }),
}))

vi.mock('@/contexts/connection', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/contexts/connection')>()),
  useConnectionPolicy: () => ({ readOnly: policy.readOnly, label: null }),
  useServerCapabilities: () => ({ unsupportedReason: () => undefined }),
}))

const mutation = { mutate: vi.fn(), mutateAsync: vi.fn(), isPending: false }

vi.mock('@/contexts/streams', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/contexts/streams')>()),
  useConsumers: () => ({ data: list.isLoading ? undefined : [], isLoading: list.isLoading, error: null, refetch: vi.fn(), isFetching: list.isLoading }),
  useStreamDetail: () => ({ data: undefined }),
  useCreateConsumer: () => mutation,
  useUpdateConsumer: () => mutation,
  useDeleteConsumer: () => mutation,
  usePauseConsumer: () => mutation,
  useResumeConsumer: () => mutation,
  useResetConsumer: () => mutation,
  useUnpinConsumer: () => mutation,
}))

function renderTab(readOnly: boolean, isLoading = false) {
  policy.readOnly = readOnly
  list.isLoading = isLoading
  render(
    <MemoryRouter>
      <StreamConsumersTab />
    </MemoryRouter>,
  )
}

describe('StreamConsumersTab with no consumer selected', () => {
  it('offers to create a consumer', () => {
    renderTab(false)
    expect(screen.getByRole('button', { name: 'Create New Consumer' })).toBeInTheDocument()
  })

  it('offers no create button on a read-only connection', () => {
    renderTab(true)
    expect(screen.queryByRole('button', { name: 'Create New Consumer' })).not.toBeInTheDocument()
  })
})

describe('StreamConsumersTab while the consumers load', () => {
  it('shows neither the empty pane nor the create offer', () => {
    renderTab(false, true)
    expect(screen.queryByText(/Select a consumer/)).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Create New Consumer' })).not.toBeInTheDocument()
  })
})
