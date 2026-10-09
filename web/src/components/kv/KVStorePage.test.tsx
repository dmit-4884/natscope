import { beforeEach, describe, expect, it, vi } from 'vitest'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { fireEvent, render, screen, waitFor } from '@/test/utils'
import KVStorePage from './KVStorePage'

const keys = vi.hoisted(() => ({
  list: { keys: ['alpha', 'beta'], truncated: false },
  filters: [] as Array<string | undefined>,
  buckets: [] as Array<{ bucket: string; values: number; bytes: number; limit_marker_ttl?: number }>,
  watch: {
    status: 'off' as string,
    error: undefined as string | undefined,
    changes: [] as Array<{ key: string; operation: string; revision: number; created: number; value: string; size: number }>,
    restart: () => {},
  },
  watchEnabled: [] as boolean[],
  revisions: {} as Record<string, number>,
}))

const entries: Record<string, { key: string; value: string; revision: number; created: number; operation: string; ttl?: number }> = {
  alpha: { key: 'alpha', value: btoa('first value'), revision: 1, created: 0, operation: 'PUT' },
  beta: { key: 'beta', value: btoa('second value'), revision: 2, created: 0, operation: 'PUT' },
  session: { key: 'session', value: btoa('s'), revision: 3, created: Date.UTC(2026, 9, 9, 12, 0, 0), operation: 'PUT', ttl: 90_000_000_000 },
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
  useKVBuckets: () => ({ data: keys.buckets }),
  useKVKeys: (_conn: string, _bucket: string, filter?: string) => {
    keys.filters.push(filter)
    return { data: keys.list, isLoading: false, error: null, refetch: vi.fn() }
  },
  useKVKey: (_conn: string, _bucket: string, key?: string) => ({
    data: key ? { ...entries[key], revision: keys.revisions[key] ?? entries[key].revision } : undefined,
    isLoading: false,
  }),
  useKVWatch: (_conn: string, _bucket: string, _filter: string, enabled: boolean) => {
    keys.watchEnabled.push(enabled)
    return enabled ? keys.watch : { ...keys.watch, status: 'off', changes: [] }
  },
  useKVKeyHistory: () => ({ data: undefined, isLoading: false, error: null }),
  usePutKVKey: () => mutation,
  useDeleteKVKey: () => mutation,
  usePurgeKVKey: () => mutation,
  useDeleteKVBucket: () => mutation,
  usePurgeKVBucket: () => mutation,
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
    keys.buckets = []
    keys.watch = { status: 'live', error: undefined, changes: [], restart: vi.fn() }
    keys.watchEnabled = []
    keys.revisions = {}
    mutation.mutateAsync.mockReset()
  })

  it('watches the bucket only after Live is turned on', () => {
    renderPage()
    expect(keys.watchEnabled.every((on) => !on)).toBe(true)

    fireEvent.click(screen.getByRole('switch', { name: /live updates/i }))

    expect(keys.watchEnabled[keys.watchEnabled.length - 1]).toBe(true)
    expect(screen.getByText(/^live$/i)).toBeInTheDocument()
  })

  it('lists the changes and opens a changed key', () => {
    keys.watch.changes = [
      { key: 'beta', operation: 'put', revision: 7, created: 0, value: btoa('second value'), size: 12 },
      { key: 'gone', operation: 'delete', revision: 6, created: 0, value: '', size: 0 },
    ]
    renderPage()
    fireEvent.click(screen.getByRole('switch', { name: /live updates/i }))

    fireEvent.click(screen.getByRole('tab', { name: /changes/i }))
    expect(screen.getByText('delete')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: /beta/ }))

    expect(screen.getByLabelText('Key value')).toHaveValue('second value')
  })

  it('offers a restart when the watch stops', () => {
    keys.watch = { ...keys.watch, status: 'stopped', error: 'connection closed' }
    renderPage()
    fireEvent.click(screen.getByRole('switch', { name: /live updates/i }))

    expect(screen.getByText(/connection closed/i)).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: /restart/i }))

    expect(keys.watch.restart).toHaveBeenCalled()
  })

  it('keeps the revision an edit started from, and says when the server moved on', async () => {
    mutation.mutateAsync.mockResolvedValue({ revision: 9 })
    const view = render(
      <MemoryRouter initialEntries={['/kv/CONFIG']}>
        <Routes>
          <Route path="/kv/:bucketName" element={<KVStorePage />} />
        </Routes>
      </MemoryRouter>,
    )
    fireEvent.click(screen.getByRole('button', { name: 'alpha' }))
    fireEvent.change(screen.getByLabelText('Key value'), { target: { value: 'my edit' } })

    keys.revisions.alpha = 5
    view.rerender(
      <MemoryRouter initialEntries={['/kv/CONFIG']}>
        <Routes>
          <Route path="/kv/:bucketName" element={<KVStorePage />} />
        </Routes>
      </MemoryRouter>,
    )

    expect(screen.getByText(/changed on the server/i)).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Save Value' }))
    await waitFor(() => expect(mutation.mutateAsync).toHaveBeenCalled())
    expect(mutation.mutateAsync.mock.calls[0][0]).toMatchObject({ key: 'alpha', expectedRevision: 1 })
  })

  it('creates a key with a TTL when the bucket allows one', async () => {
    keys.buckets = [{ bucket: 'CONFIG', values: 0, bytes: 0, limit_marker_ttl: 1_000_000_000 }]
    mutation.mutateAsync.mockResolvedValue({ revision: 1 })
    renderPage()

    fireEvent.click(screen.getByRole('button', { name: 'New key' }))
    fireEvent.change(screen.getByPlaceholderText('my.key.name'), { target: { value: 'session.1' } })
    fireEvent.change(screen.getByLabelText(/^ttl$/i), { target: { value: '1m30s' } })
    fireEvent.click(screen.getByRole('button', { name: 'Create Key' }))

    await waitFor(() => expect(mutation.mutateAsync).toHaveBeenCalled())
    expect(mutation.mutateAsync.mock.calls[0][0]).toMatchObject({ key: 'session.1', ttl: 90_000_000_000 })
  })

  it('refuses a TTL it cannot read', () => {
    keys.buckets = [{ bucket: 'CONFIG', values: 0, bytes: 0, limit_marker_ttl: 1_000_000_000 }]
    renderPage()

    fireEvent.click(screen.getByRole('button', { name: 'New key' }))
    fireEvent.change(screen.getByPlaceholderText('my.key.name'), { target: { value: 'session.1' } })
    fireEvent.change(screen.getByLabelText(/^ttl$/i), { target: { value: 'soon' } })

    expect(screen.getByText(/duration like 30s/i)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Create Key' })).toBeDisabled()
  })

  it('points to the bucket settings when the bucket allows no TTL per key', () => {
    renderPage()

    fireEvent.click(screen.getByRole('button', { name: 'New key' }))

    expect(screen.queryByLabelText(/^ttl$/i)).not.toBeInTheDocument()
    expect(screen.getByText(/key ttl marker/i)).toBeInTheDocument()
  })

  it('warns that saving a key with a TTL drops the TTL', () => {
    keys.list = { keys: ['session', 'alpha'], truncated: false }
    renderPage()

    fireEvent.click(screen.getByRole('button', { name: 'alpha' }))
    expect(screen.queryByText(/stop expiring/i)).not.toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'session' }))
    expect(screen.getByText(/stop expiring/i)).toBeInTheDocument()
  })

  it('shows when a key with a TTL expires', () => {
    keys.list = { keys: ['session'], truncated: false }
    renderPage()

    fireEvent.click(screen.getByRole('button', { name: 'session' }))

    expect(screen.getByText(/expires/i)).toBeInTheDocument()
    expect(screen.getByText(/1m 30s|1m30s|90s/i)).toBeInTheDocument()
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

  it('clears the bucket after its name is typed, warning that watchers are not told', async () => {
    mutation.mutateAsync.mockResolvedValue(undefined)
    renderPage()

    fireEvent.click(screen.getByRole('button', { name: 'Bucket actions' }))
    fireEvent.click(await screen.findByRole('menuitem', { name: /clear bucket/i }))

    expect(screen.getByText(/watching the bucket are not told/i)).toBeInTheDocument()
    const confirm = screen.getByRole('button', { name: 'Clear KV Store' })
    expect(confirm).toBeDisabled()
    fireEvent.change(screen.getByPlaceholderText('CONFIG'), { target: { value: 'CONFIG' } })
    fireEvent.click(confirm)

    await waitFor(() => expect(mutation.mutateAsync).toHaveBeenCalledWith('CONFIG'))
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
