import { describe, it, expect, vi } from 'vitest'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { render, screen } from '@/test/utils'
import ObjectsTab from './ObjectsTab'

const policy = vi.hoisted(() => ({ readOnly: false }))

vi.mock('react-router-dom', async (importOriginal) => ({
  ...(await importOriginal<typeof import('react-router-dom')>()),
  useOutletContext: () => ({ connectionId: 'conn-1', currentConnection: null }),
}))

vi.mock('@/contexts/connection', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/contexts/connection')>()),
  useConnectionPolicy: () => ({ readOnly: policy.readOnly, label: null }),
}))

const mutation = { mutate: vi.fn(), mutateAsync: vi.fn(), isPending: false }
const bucket = { bucket: 'files', description: '', storage: 'file', replicas: 1, size: 1, sealed: false, ttl: 0 }
const object = { name: 'report.txt', nuid: 'n1', size: 1, chunks: 1, mtime: 0, bucket: 'files', digest: '', deleted: false }

vi.mock('@/contexts/objects', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/contexts/objects')>()),
  useObjectBuckets: () => ({ data: [bucket], refetch: vi.fn() }),
  useObjects: () => ({ data: [object], isLoading: false, error: null, refetch: vi.fn() }),
  useObject: () => ({ data: undefined, isLoading: false }),
  useCreateObjectBucket: () => mutation,
  useDeleteObjectBucket: () => mutation,
  useSealObjectBucket: () => mutation,
  usePutObject: () => mutation,
  useDeleteObject: () => mutation,
}))

function renderTab(readOnly: boolean) {
  policy.readOnly = readOnly
  render(
    <MemoryRouter initialEntries={['/objects/files']}>
      <Routes>
        <Route path="/objects/:bucketName" element={<ObjectsTab />} />
      </Routes>
    </MemoryRouter>,
  )
}

describe('ObjectsTab objects', () => {
  it('lets a writable connection delete an object', () => {
    renderTab(false)
    expect(screen.getByRole('button', { name: 'Delete object report.txt' })).toBeInTheDocument()
  })

  it('offers no delete on a read-only connection', () => {
    renderTab(true)
    expect(screen.getByText('report.txt')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Delete object report.txt' })).not.toBeInTheDocument()
  })
})
