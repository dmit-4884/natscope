import { beforeEach, describe, expect, it, vi } from 'vitest'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { fireEvent, render, screen, waitFor } from '@/test/utils'
import KVStorePage from './KVStorePage'

const keys = vi.hoisted(() => ({
  list: { keys: ['alpha', 'beta'], truncated: false },
  filters: [] as Array<string | undefined>,
}))

const entries: Record<string, { key: string; value: string; revision: number; created: number; operation: string }> = {
  alpha: { key: 'alpha', value: btoa('first value'), revision: 1, created: 0, operation: 'PUT' },
  beta: { key: 'beta', value: btoa('second value'), revision: 2, created: 0, operation: 'PUT' },
}

const mutation = { mutate: vi.fn(), mutateAsync: vi.fn(), isPending: false }

vi.mock('react-router-dom', async (importOriginal) => ({
  ...(await importOriginal<typeof import('react-router-dom')>()),
  useOutletContext: () => ({ connectionId: 'conn-1', currentConnection: null }),
}))

vi.mock('@/contexts/connection', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/contexts/connection')>()),
  useConnectionPolicy: () => ({ readOnly: false, label: null, known: true }),
}))

vi.mock('@/contexts/kv', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/contexts/kv')>()),
  useKVBuckets: () => ({ data: [] }),
  useKVKeys: (_conn: string, _bucket: string, filter?: string) => {
    keys.filters.push(filter)
    return { data: keys.list, isLoading: false, error: null, refetch: vi.fn() }
  },
  useKVKey: (_conn: string, _bucket: string, key?: string) => ({ data: key ? entries[key] : undefined, isLoading: false }),
  useKVKeyHistory: () => ({ data: undefined, isLoading: false, error: null }),
  usePutKVKey: () => mutation,
  useDeleteKVKey: () => mutation,
  usePurgeKVKey: () => mutation,
  useDeleteKVBucket: () => mutation,
}))

vi.mock('./useKVProtoTarget', () => ({ useKVProtoTarget: () => null }))

function renderPage() {
  render(
    <MemoryRouter initialEntries={['/kv/CONFIG']}>
      <Routes>
        <Route path="/kv/:bucketName" element={<KVStorePage />} />
        <Route path="/kv/:bucketName/edit" element={<p>edit page</p>} />
      </Routes>
    </MemoryRouter>,
  )
}

describe('KVStorePage', () => {
  beforeEach(() => {
    keys.list = { keys: ['alpha', 'beta'], truncated: false }
    keys.filters = []
  })

  it('sends a wildcard pattern to the server', async () => {
    renderPage()

    fireEvent.change(screen.getByPlaceholderText(/search keys/i), { target: { value: 'orders.*' } })

    await waitFor(() => expect(keys.filters[keys.filters.length - 1]).toBe('orders.*'))
  })

  it('filters loaded keys by plain text without asking the server', async () => {
    renderPage()

    fireEvent.change(screen.getByPlaceholderText(/search keys/i), { target: { value: 'alp' } })

    await waitFor(() => expect(screen.queryByRole('button', { name: 'beta' })).not.toBeInTheDocument())
    expect(screen.getByRole('button', { name: 'alpha' })).toBeInTheDocument()
    expect(keys.filters.every((f) => !f)).toBe(true)
  })

  it('opens the bucket editor from the bucket menu', async () => {
    renderPage()

    fireEvent.click(screen.getByRole('button', { name: 'Bucket actions' }))
    fireEvent.click(await screen.findByRole('menuitem', { name: /edit bucket/i }))

    expect(await screen.findByText('edit page')).toBeInTheDocument()
  })

  it('says when the server cut the list at the limit', () => {
    keys.list = { keys: ['alpha', 'beta'], truncated: true }

    renderPage()

    expect(screen.getByText(/first 2 keys/i)).toBeInTheDocument()
    expect(screen.getByText(/narrow/i)).toBeInTheDocument()
  })

  it('shows each picked key with its own value, also one already loaded before', () => {
    renderPage()
    fireEvent.click(screen.getByRole('button', { name: 'alpha' }))
    expect(screen.getByLabelText('Key value')).toHaveValue('first value')

    fireEvent.click(screen.getByRole('button', { name: 'beta' }))

    expect(screen.getByLabelText('Key value')).toHaveValue('second value')
  })
})
