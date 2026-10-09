import { describe, it, expect, vi, beforeEach } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import { render, screen, fireEvent, waitFor } from '@/test/utils'
import { getSidebarLayout, updateSidebarLayout, type SidebarLayout } from '@/api/connections'
import { resetSidebarUi } from '@/stores/sidebarUiStore'
import { SidebarResourceList } from './SidebarResourceList'

vi.mock('@/api/connections', () => ({
  getSidebarLayout: vi.fn(),
  updateSidebarLayout: vi.fn(),
}))

vi.mock('@/utils/toast', () => ({
  toast: { success: vi.fn(), error: vi.fn(), info: vi.fn(), warning: vi.fn() },
}))

const mockedGet = vi.mocked(getSidebarLayout)
const mockedUpdate = vi.mocked(updateSidebarLayout)

function layout(streams: Partial<SidebarLayout['streams']> = {}): SidebarLayout {
  return {
    streams: { pinned: [], order: [], ...streams },
    kv: { pinned: [], order: [] },
    objects: { pinned: [], order: [] },
  }
}

function renderList(names: string[], selectedName?: string) {
  return render(
    <MemoryRouter>
      <SidebarResourceList
        connectionId="conn-1"
        section="streams"
        names={names}
        selectedName={selectedName}
        hrefFor={(name) => `/streams/${name}`}
        noun="stream"
        isRefreshing={false}
        onRefresh={vi.fn()}
      />
    </MemoryRouter>,
  )
}

const itemNames = () => screen.getAllByTestId('sidebar-item').map((li) => li.querySelector('a')?.textContent)
const lastPatch = () => mockedUpdate.mock.lastCall?.[1]

describe('SidebarResourceList', () => {
  beforeEach(() => {
    resetSidebarUi()
    mockedGet.mockReset()
    mockedUpdate.mockReset()
    mockedUpdate.mockImplementation(async (_id, patch) => ({ ...layout(), ...patch }))
  })

  it('lists pinned names first, then the manual order, then the rest by name', async () => {
    mockedGet.mockResolvedValue(layout({ pinned: ['zeta'], order: ['orders'] }))
    renderList(['alpha', 'orders', 'beta', 'zeta'])

    await waitFor(() => expect(itemNames()).toEqual(['zeta', 'orders', 'alpha', 'beta']))
    expect(screen.getByText('Pinned')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Unpin zeta' })).toHaveAttribute('aria-pressed', 'true')
  })

  it('filters by name', async () => {
    mockedGet.mockResolvedValue(layout())
    renderList(['orders', 'order_audit', 'events'])

    fireEvent.change(screen.getByRole('textbox', { name: 'Filter streams' }), { target: { value: 'ORDER' } })

    expect(itemNames()).toEqual(['order_audit', 'orders'])
  })

  it('keeps the filter when the sidebar is shown again', () => {
    mockedGet.mockResolvedValue(layout())
    const first = renderList(['orders', 'events'])
    fireEvent.change(screen.getByRole('textbox', { name: 'Filter streams' }), { target: { value: 'ord' } })
    first.unmount()

    renderList(['orders', 'events'])

    expect(screen.getByRole('textbox', { name: 'Filter streams' })).toHaveValue('ord')
  })

  it('says when nothing matches', async () => {
    mockedGet.mockResolvedValue(layout())
    renderList(['orders'])

    fireEvent.change(screen.getByRole('textbox', { name: 'Filter streams' }), { target: { value: 'nope' } })

    expect(screen.getByText('No streams match “nope”')).toBeInTheDocument()
  })

  it('shows 12 first, then 50 more at a time', async () => {
    mockedGet.mockResolvedValue(layout())
    const names = Array.from({ length: 70 }, (_, i) => `s${String(i).padStart(3, '0')}`)
    renderList(names)

    expect(itemNames()).toHaveLength(12)
    expect(screen.getByTestId('sidebar-show-more')).toHaveTextContent('Show 50 more (58 hidden)')
    fireEvent.click(screen.getByTestId('sidebar-show-more'))
    expect(itemNames()).toHaveLength(62)
    expect(screen.getByTestId('sidebar-show-more')).toHaveTextContent('Show 8 more (8 hidden)')
    fireEvent.click(screen.getByTestId('sidebar-show-more'))
    expect(itemNames()).toHaveLength(70)
    expect(screen.queryByTestId('sidebar-show-more')).toBeNull()
  })

  it('keeps the selected item visible beyond the first 12', async () => {
    mockedGet.mockResolvedValue(layout())
    const names = Array.from({ length: 40 }, (_, i) => `s${String(i).padStart(3, '0')}`)
    renderList(names, 's029')

    const visible = itemNames()
    expect(visible).toHaveLength(30)
    expect(visible[visible.length - 1]).toBe('s029')
  })

  it('pins with the star', async () => {
    mockedGet.mockResolvedValue(layout({ order: ['beta', 'alpha'] }))
    renderList(['alpha', 'beta'])
    await waitFor(() => expect(itemNames()).toEqual(['beta', 'alpha']))

    fireEvent.click(screen.getByRole('button', { name: 'Pin alpha' }))

    await waitFor(() => expect(lastPatch()).toEqual({ streams: { pinned: ['alpha'], order: ['beta'] } }))
    await waitFor(() => expect(itemNames()).toEqual(['alpha', 'beta']))
  })

  it('moves an item with Alt+ArrowDown', async () => {
    mockedGet.mockResolvedValue(layout())
    renderList(['alpha', 'beta', 'gamma'])
    await waitFor(() => expect(mockedGet).toHaveBeenCalled())

    fireEvent.keyDown(screen.getByRole('link', { name: 'alpha' }), { key: 'ArrowDown', altKey: true })

    await waitFor(() => expect(lastPatch()).toEqual({ streams: { pinned: [], order: ['beta', 'alpha', 'gamma'] } }))
    await waitFor(() => expect(itemNames()).toEqual(['beta', 'alpha', 'gamma']))
    expect(screen.getByRole('status')).toHaveTextContent('alpha moved to position 2 of 3')
  })

  it('reorders by drag and drop', async () => {
    mockedGet.mockResolvedValue(layout())
    renderList(['alpha', 'beta', 'gamma'])
    await waitFor(() => expect(mockedGet).toHaveBeenCalled())

    const [alpha, , gamma] = screen.getAllByTestId('sidebar-item')
    vi.spyOn(alpha, 'getBoundingClientRect').mockReturnValue({ top: 100, height: 20 } as DOMRect)
    const dataTransfer = { setData: vi.fn(), effectAllowed: '', dropEffect: '' }

    fireEvent.dragStart(gamma, { dataTransfer })
    fireEvent.dragOver(alpha, { dataTransfer, clientY: 101 })
    fireEvent.drop(alpha, { dataTransfer })

    await waitFor(() => expect(lastPatch()).toEqual({ streams: { pinned: [], order: ['gamma', 'alpha', 'beta'] } }))
  })

  it('offers a reset back to A–Z once there is a manual order', async () => {
    mockedGet.mockResolvedValue(layout({ pinned: ['beta'], order: ['gamma', 'alpha'] }))
    renderList(['alpha', 'beta', 'gamma'])
    await waitFor(() => expect(itemNames()).toEqual(['beta', 'gamma', 'alpha']))

    fireEvent.click(screen.getByRole('button', { name: 'Stream list options' }))
    fireEvent.click(screen.getByRole('menuitem', { name: 'Reset to A–Z order' }))

    await waitFor(() => expect(lastPatch()).toEqual({ streams: { pinned: ['beta'], order: [] } }))
  })

  it('does not reorder while filtering', async () => {
    mockedGet.mockResolvedValue(layout())
    renderList(['alpha', 'beta'])
    await waitFor(() => expect(mockedGet).toHaveBeenCalled())

    fireEvent.change(screen.getByRole('textbox', { name: 'Filter streams' }), { target: { value: 'a' } })
    fireEvent.keyDown(screen.getByRole('link', { name: 'alpha' }), { key: 'ArrowDown', altKey: true })

    expect(mockedUpdate).not.toHaveBeenCalled()
    expect(screen.getAllByTestId('sidebar-item')[0]).not.toHaveAttribute('draggable')
  })
})
