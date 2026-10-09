import { describe, it, expect, vi } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import { render, screen } from '@/test/utils'
import type { ConsumerInfo } from '@/types/nats'
import StreamConsumersTab from './StreamConsumersTab'

const policy = vi.hoisted(() => ({ readOnly: false }))
const list = vi.hoisted(() => ({ isLoading: false, data: [] as ConsumerInfo[], updatedAt: 0, refetch: vi.fn() }))

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
  useConsumers: () => ({
    data: list.isLoading ? undefined : list.data,
    isLoading: list.isLoading,
    error: null,
    refetch: list.refetch,
    isFetching: list.isLoading,
    dataUpdatedAt: list.updatedAt,
    errorUpdatedAt: 0,
  }),
  useStreamDetail: () => ({ data: undefined }),
  useCreateConsumer: () => mutation,
  useUpdateConsumer: () => mutation,
  useDeleteConsumer: () => mutation,
  usePauseConsumer: () => mutation,
  useResumeConsumer: () => mutation,
  useResetConsumer: () => mutation,
  useUnpinConsumer: () => mutation,
}))

vi.mock('@/stores/streamTabState/consumerEditorStore', async () => {
  const { useCallback, useState } = await import('react')
  const defaults = { selectedName: null, searchQuery: '', isCreating: false, isEditing: false, editorMode: 'form', formDraft: null }
  return {
    useConsumerEditorEntry: () => {
      const [state, setState] = useState<Record<string, unknown>>(defaults)
      const update = useCallback((partial: Record<string, unknown>) => setState((prev) => ({ ...prev, ...partial })), [])
      return [state, update]
    },
  }
})

vi.mock('./consumers/ConsumerView', () => ({
  ConsumerView: ({ consumer }: { consumer: ConsumerInfo }) => <h2>{consumer.name}</h2>,
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

const tabAt = (entry: string) => (
  <MemoryRouter initialEntries={[entry]}>
    <StreamConsumersTab />
  </MemoryRouter>
)

const consumer = (name: string) =>
  ({ name, num_pending: 0, num_ack_pending: 0, config: { deliver_policy: 'all', ack_policy: 'explicit' } }) as ConsumerInfo

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

describe('StreamConsumersTab opened through a consumer link', () => {
  it('waits for a fresh list when the cached one lacks the linked consumer', () => {
    policy.readOnly = false
    list.isLoading = false
    list.data = [consumer('worker')]
    list.updatedAt = 1
    list.refetch.mockClear()
    const view = render(tabAt('/?consumer=audit'))

    expect(list.refetch).toHaveBeenCalled()
    expect(screen.queryByText(/Select a consumer/)).not.toBeInTheDocument()

    list.data = [consumer('worker'), consumer('audit')]
    list.updatedAt = 2
    view.rerender(tabAt('/?consumer=audit'))
    expect(screen.getByRole('heading', { name: 'audit' })).toBeInTheDocument()
  })

  it('lets go of a linked consumer the fresh list does not have', () => {
    policy.readOnly = false
    list.isLoading = false
    list.data = [consumer('worker')]
    list.updatedAt = 1
    const view = render(tabAt('/?consumer=gone'))

    list.updatedAt = 2
    view.rerender(tabAt('/?consumer=gone'))
    expect(screen.getByText(/Select a consumer/)).toBeInTheDocument()
  })
})
