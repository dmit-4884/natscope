import { describe, expect, it, vi } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import { render } from '@/test/utils'
import KVList from '@/components/kv/KVList'
import ObjectList from '@/components/objects/ObjectList'
import StreamList from '@/components/streams/StreamList'

const loading = { data: undefined, isLoading: true, error: null, refetch: vi.fn(), isFetching: true }
const loaded = { data: ['ORDERS'], isLoading: false, error: null, refetch: vi.fn(), isFetching: false }
const state = vi.hoisted(() => ({ namesLoaded: false, layoutPending: false }))

vi.mock('@/contexts/connection', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/contexts/connection')>()),
  useConnectionPolicy: () => ({ readOnly: false, label: null }),
  useSidebarLayoutPending: () => state.layoutPending,
}))

vi.mock('@/contexts/streams', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/contexts/streams')>()),
  useStreamNames: () => (state.namesLoaded ? loaded : loading),
}))

vi.mock('@/contexts/objects', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/contexts/objects')>()),
  useObjectBuckets: () => loading,
}))

describe('sidebar lists while loading', () => {
  it.each([
    ['streams', StreamList],
    ['KV stores', KVList],
    ['object stores', ObjectList],
  ])('hold one row for %s, so the section grows into its list instead of shrinking to it', (_, List) => {
    const { container } = render(
      <MemoryRouter>
        <List connectionId="conn-1" />
      </MemoryRouter>,
    )
    expect(container.querySelectorAll('.animate-shimmer')).toHaveLength(1)
  })
})

describe('sidebar lists before the pins and order arrive', () => {
  it('hold the loading row instead of listing names A–Z first', () => {
    state.namesLoaded = true
    state.layoutPending = true
    const { container } = render(
      <MemoryRouter>
        <StreamList connectionId="conn-1" />
      </MemoryRouter>,
    )

    expect(container.querySelectorAll('.animate-shimmer')).toHaveLength(1)
    state.namesLoaded = false
    state.layoutPending = false
  })
})
