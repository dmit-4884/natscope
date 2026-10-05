import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom'
import StreamTabs from './StreamTabs'

const policy = vi.hoisted(() => ({ readOnly: false }))

vi.mock('@/contexts/connection', () => ({
  useConnectionPolicy: () => ({ readOnly: policy.readOnly, label: null }),
}))

const TAB_WIDTH = 120
let rowWidth = 0

function Location() {
  return <div data-testid="location">{useLocation().pathname}</div>
}

function renderAt(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route
          path="/streams/:streamName/*"
          element={
            <>
              <StreamTabs baseUrl="/streams/ORDERS" />
              <Location />
            </>
          }
        />
      </Routes>
    </MemoryRouter>,
  )
}

const tabNames = () => screen.getAllByRole('link').map((l) => l.textContent)

describe('StreamTabs', () => {
  beforeEach(() => {
    vi.spyOn(HTMLElement.prototype, 'clientWidth', 'get').mockImplementation(function (this: HTMLElement) {
      return this.dataset.testid === 'stream-tabs' ? rowWidth : 0
    })
    vi.spyOn(HTMLElement.prototype, 'offsetWidth', 'get').mockImplementation(function (this: HTMLElement) {
      return this.dataset.tab ? TAB_WIDTH : 0
    })
  })

  afterEach(() => {
    vi.restoreAllMocks()
    policy.readOnly = false
  })

  it('drops the Publish tab on a read-only connection', () => {
    rowWidth = 1000
    policy.readOnly = true
    renderAt('/streams/ORDERS/messages')

    expect(tabNames()).toEqual(['Messages', 'Config', 'Consumers', 'Relations'])
  })

  it('shows every tab when they fit', () => {
    rowWidth = 1000
    renderAt('/streams/ORDERS/messages')

    expect(tabNames()).toEqual(['Messages', 'Config', 'Consumers', 'Relations', 'Publish'])
    expect(screen.queryByRole('button', { name: 'More tabs' })).not.toBeInTheDocument()
  })

  it('moves the tabs that do not fit behind an arrow menu', () => {
    rowWidth = 300
    renderAt('/streams/ORDERS/messages')

    expect(tabNames()).toEqual(['Messages', 'Config'])
    fireEvent.click(screen.getByRole('button', { name: 'More tabs' }))
    expect(screen.getAllByRole('menuitem').map((i) => i.textContent)).toEqual(['Consumers', 'Relations', 'Publish'])
  })

  it('marks the arrow and the menu entry when the active tab is hidden', () => {
    rowWidth = 300
    renderAt('/streams/ORDERS/publish')

    const more = screen.getByRole('button', { name: 'More tabs' })
    expect(more).toHaveClass('text-accent')
    fireEvent.click(more)
    expect(screen.getByRole('menuitem', { name: 'Publish' })).toHaveAttribute('aria-current', 'page')
  })

  it('navigates to a hidden tab from the menu', () => {
    rowWidth = 300
    renderAt('/streams/ORDERS/messages')

    fireEvent.click(screen.getByRole('button', { name: 'More tabs' }))
    fireEvent.click(screen.getByRole('menuitem', { name: 'Relations' }))

    expect(screen.getByTestId('location')).toHaveTextContent('/streams/ORDERS/relations')
  })
})

describe('StreamTabs on a resize', () => {
  const resized: Array<() => void> = []

  beforeEach(() => {
    vi.spyOn(HTMLElement.prototype, 'clientWidth', 'get').mockImplementation(function (this: HTMLElement) {
      return this.dataset.testid === 'stream-tabs' ? rowWidth : 0
    })
    vi.spyOn(HTMLElement.prototype, 'offsetWidth', 'get').mockImplementation(function (this: HTMLElement) {
      return this.dataset.tab ? TAB_WIDTH : 0
    })
    vi.stubGlobal(
      'ResizeObserver',
      class {
        constructor(callback: () => void) {
          resized.push(callback)
        }
        observe() {}
        disconnect() {}
      },
    )
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
    resized.length = 0
  })

  it('refits before the next paint, so the row never shows the old set of tabs', () => {
    rowWidth = 1000
    renderAt('/streams/ORDERS/messages')
    expect(screen.queryByRole('button', { name: 'More tabs' })).not.toBeInTheDocument()

    rowWidth = 300
    vi.spyOn(console, 'error').mockImplementation(() => {})
    resized.forEach((callback) => callback())

    expect(tabNames()).toEqual(['Messages', 'Config'])
  })
})
