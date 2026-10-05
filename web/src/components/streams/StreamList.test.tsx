import { describe, it, expect, vi } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import { render, screen } from '@/test/utils'
import StreamList from './StreamList'

const policy = vi.hoisted(() => ({ readOnly: false }))

vi.mock('@/contexts/connection', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/contexts/connection')>()),
  useConnectionPolicy: () => ({ readOnly: policy.readOnly, label: null }),
}))

vi.mock('@/contexts/streams', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/contexts/streams')>()),
  useStreamNames: () => ({ data: [], isLoading: false, error: null, refetch: vi.fn(), isFetching: false }),
}))

function renderList(readOnly: boolean) {
  policy.readOnly = readOnly
  render(
    <MemoryRouter>
      <StreamList connectionId="conn-1" />
    </MemoryRouter>,
  )
}

describe('StreamList without streams', () => {
  it('offers to create a stream', () => {
    renderList(false)
    expect(screen.getByRole('link', { name: 'Create stream' })).toBeInTheDocument()
  })

  it('offers no create link on a read-only connection', () => {
    renderList(true)
    expect(screen.queryByRole('link', { name: 'Create stream' })).not.toBeInTheDocument()
    expect(screen.getByText('No streams')).toBeInTheDocument()
  })
})
