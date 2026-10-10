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
  useUploadObject: () => mutation,
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


describe('ObjectsTab transfers', () => {
  beforeEach(() => {
    fetched.names = []
    fetched.objects = [{ ...object, name: 'dump.bin', size: 200 * 1024 * 1024, mod_time: 0 }]
    mutation.mutateAsync.mockReset()
  })

  it('downloads an object of any size and only skips the preview of a large one', () => {
    renderTab(false)

    fireEvent.click(screen.getByText('dump.bin'))

    const link = screen.getByRole('link', { name: /download/i })
    expect(Object.fromEntries(new URL(link.getAttribute('href')!).searchParams)).toEqual({
      connection: 'conn-1',
      bucket: 'files',
      name: 'dump.bin',
    })
    expect(link).toHaveAttribute('download', 'dump.bin')
    expect(screen.getByText(/too large to preview/i)).toBeInTheDocument()
    expect(fetched.names.filter(Boolean)).toEqual([])
  })

  it('uploads a file of any size as it is', () => {
    renderTab(false)
    const file = new File(['x'], 'huge.bin')
    Object.defineProperty(file, 'size', { value: 300 * 1024 * 1024 })

    fireEvent.change(document.querySelector('input[type="file"]')!, { target: { files: [file] } })

    expect(mutation.mutateAsync).toHaveBeenCalledWith({ file, description: undefined })
    expect(vi.mocked(toast.error)).not.toHaveBeenCalled()
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
