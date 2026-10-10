import { beforeEach, describe, it, expect, vi } from 'vitest'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { fireEvent, render, screen } from '@/test/utils'
import { toast } from '@/utils/toast'
import ObjectsTab from './ObjectsTab'

const policy = vi.hoisted(() => ({ readOnly: false }))
const fetched = vi.hoisted(() => ({ names: [] as Array<string | undefined>, objects: [] as unknown[] }))

vi.mock('@/utils/toast', () => ({ toast: { error: vi.fn(), success: vi.fn() } }))

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
  useObjects: () => ({ data: fetched.objects, isLoading: false, error: null, refetch: vi.fn() }),
  useObject: (_conn: string, _bucket: string, name?: string) => {
    fetched.names.push(name)
    return { data: undefined, isLoading: false }
  },
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

describe('ObjectsTab transfer limit', () => {
  beforeEach(() => {
    fetched.names = []
    fetched.objects = [{ ...object, name: 'dump.bin', size: 200 * 1024 * 1024, mod_time: 0 }]
    mutation.mutateAsync.mockReset()
  })

  it('says an object over the limit is too big to open here and how to get it', () => {
    renderTab(false)

    fireEvent.click(screen.getByText('dump.bin'))

    expect(screen.getByText(/over 32 MiB/i)).toBeInTheDocument()
    expect(screen.getByText('nats object get files dump.bin')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /download/i })).not.toBeInTheDocument()
    expect(fetched.names.filter(Boolean)).toEqual([])
  })

  it('refuses a file over the limit before reading it', () => {
    renderTab(false)
    const file = new File(['x'], 'huge.bin')
    Object.defineProperty(file, 'size', { value: 300 * 1024 * 1024 })

    fireEvent.change(document.querySelector('input[type="file"]')!, { target: { files: [file] } })

    expect(vi.mocked(toast.error)).toHaveBeenCalledWith(expect.stringMatching(/huge\.bin.*32 MiB/))
    expect(mutation.mutateAsync).not.toHaveBeenCalled()
  })
})

describe('ObjectsTab objects', () => {
  beforeEach(() => {
    fetched.objects = [object]
  })

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
