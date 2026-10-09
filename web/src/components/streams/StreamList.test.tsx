import { describe, it, expect, vi } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import { render, screen } from '@/test/utils'
import StreamList from './StreamList'

const policy = vi.hoisted(() => ({ readOnly: false }))
const names = vi.hoisted(() => ({ value: [] as string[] }))

vi.mock('@/contexts/connection', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/contexts/connection')>()),
  useConnectionPolicy: () => ({ readOnly: policy.readOnly, label: null }),
  useSidebarLayoutPending: () => false,
}))

vi.mock('@/contexts/streams', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/contexts/streams')>()),
  useStreamNames: () => ({ data: names.value, isLoading: false, error: null, refetch: vi.fn(), isFetching: false }),
}))

vi.mock('@/components/common/sidebar/SidebarResourceList', () => ({
  SidebarResourceList: ({ names: shown, hrefFor }: { names: string[]; hrefFor: (name: string) => string }) => (
    <ul>
      {shown.map((name) => (
        <li key={name}>
          <a href={hrefFor(name)}>{name}</a>
        </li>
      ))}
    </ul>
  ),
}))

function renderList(readOnly: boolean, streams: string[] = []) {
  policy.readOnly = readOnly
  names.value = streams
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

describe('StreamList with streams', () => {
  it('opens a stream straight on its messages, with no redirect in between', () => {
    renderList(false, ['ORDERS'])
    expect(screen.getByRole('link', { name: 'ORDERS' })).toHaveAttribute('href', '/streams/ORDERS/messages')
  })
})
