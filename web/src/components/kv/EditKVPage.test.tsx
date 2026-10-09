import { beforeEach, describe, expect, it, vi } from 'vitest'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { fireEvent, render, screen, waitFor } from '@/test/utils'
import type { KVBucketInfo } from '@/types/management'
import EditKVPage from './EditKVPage'

const state = vi.hoisted(() => ({
  bucket: null as KVBucketInfo | null,
  update: vi.fn(),
}))

vi.mock('react-router-dom', async (importOriginal) => ({
  ...(await importOriginal<typeof import('react-router-dom')>()),
  useOutletContext: () => ({ connectionId: 'conn-1', currentConnection: null }),
}))

vi.mock('@/contexts/connection', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/contexts/connection')>()),
  useServerCapabilities: () => ({ isSupported: () => true, unsupportedReason: () => undefined }),
}))

vi.mock('@/contexts/kv', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/contexts/kv')>()),
  useKVBuckets: () => ({ data: state.bucket ? [state.bucket] : [], isLoading: false, error: null }),
  useUpdateKVBucket: () => ({ mutateAsync: state.update, isPending: false }),
}))

const baseBucket: KVBucketInfo = {
  bucket: 'CONFIG',
  values: 3,
  bytes: 120,
  history: 1,
  storage: 'file',
  num_replicas: 1,
  max_value_size: -1,
  max_bytes: -1,
}

function renderPage() {
  render(
    <MemoryRouter initialEntries={['/kv/CONFIG/edit']}>
      <Routes>
        <Route path="/kv/:bucketName/edit" element={<EditKVPage />} />
        <Route path="/kv/:bucketName" element={<p>bucket page</p>} />
      </Routes>
    </MemoryRouter>,
  )
}

describe('EditKVPage', () => {
  beforeEach(() => {
    state.bucket = { ...baseBucket }
    state.update = vi.fn().mockResolvedValue(baseBucket)
  })

  it('fills the form from the bucket and keeps the name fixed', () => {
    state.bucket = { ...baseBucket, history: 3, description: 'flags' }

    renderPage()

    expect(screen.getByDisplayValue('CONFIG')).toBeDisabled()
    expect(screen.getByLabelText(/history/i)).toHaveValue(3)
    expect(screen.getByLabelText(/description/i)).toHaveValue('flags')
  })

  it('shows the changes, then saves only the editable settings and returns to the bucket', async () => {
    renderPage()

    fireEvent.change(screen.getByLabelText(/history/i), { target: { value: '5' } })
    fireEvent.click(screen.getByRole('button', { name: /save changes/i }))
    fireEvent.click(await screen.findByRole('button', { name: /confirm changes/i }))

    await waitFor(() => expect(state.update).toHaveBeenCalledTimes(1))
    const [arg] = state.update.mock.calls[0]
    expect(arg.bucket).toBe('CONFIG')
    expect(arg.settings.history).toBe(5)
    expect(arg.settings).not.toHaveProperty('storage')
    expect(arg.settings).not.toHaveProperty('bucket')
    expect(await screen.findByText('bucket page')).toBeInTheDocument()
  })

  it('does not let a per-key TTL be turned off once the bucket allows it', () => {
    state.bucket = { ...baseBucket, limit_marker_ttl: 5_000_000_000 }

    renderPage()
    fireEvent.change(screen.getByLabelText(/key ttl marker/i), { target: { value: '0' } })

    expect(screen.getByText(/can't be turned off once the bucket allows it/i)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /save changes/i })).toBeDisabled()
  })

  it('says when the bucket does not exist', () => {
    state.bucket = null

    renderPage()

    expect(screen.getByText(/not found/i)).toBeInTheDocument()
  })
})
