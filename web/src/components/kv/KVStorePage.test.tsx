import { describe, expect, it, vi } from 'vitest'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { fireEvent, render, screen } from '@/test/utils'
import KVStorePage from './KVStorePage'

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
  useKVKeys: () => ({ data: ['alpha', 'beta'], isLoading: false, error: null, refetch: vi.fn() }),
  useKVKey: (_conn: string, _bucket: string, key?: string) => ({ data: key ? entries[key] : undefined, isLoading: false }),
  useKVKeyHistory: () => ({ data: undefined, isLoading: false, error: null }),
  usePutKVKey: () => mutation,
  useDeleteKVKey: () => mutation,
  usePurgeKVKey: () => mutation,
  useDeleteKVBucket: () => mutation,
}))

vi.mock('./useKVProtoTarget', () => ({ useKVProtoTarget: () => null }))

describe('KVStorePage', () => {
  it('shows each picked key with its own value, also one already loaded before', () => {
    render(
      <MemoryRouter initialEntries={['/kv/CONFIG']}>
        <Routes>
          <Route path="/kv/:bucketName" element={<KVStorePage />} />
        </Routes>
      </MemoryRouter>,
    )
    fireEvent.click(screen.getByRole('button', { name: 'alpha' }))
    expect(screen.getByLabelText('Key value')).toHaveValue('first value')

    fireEvent.click(screen.getByRole('button', { name: 'beta' }))

    expect(screen.getByLabelText('Key value')).toHaveValue('second value')
  })
})
