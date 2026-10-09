import { afterEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { Dropdown } from './Dropdown'

const options = [
  { value: 'relative', label: 'Relative' },
  { value: 'absolute', label: 'Absolute' },
  { value: 'iso', label: 'At most 25 msg/s' },
]

function nearScrollerBottom() {
  vi.spyOn(HTMLElement.prototype, 'scrollHeight', 'get').mockReturnValue(66)
  vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockImplementation(function (this: HTMLElement) {
    if (this.dataset.testid === 'scroller') return { top: 0, bottom: 705 } as DOMRect
    if (this.tagName === 'BUTTON') return { top: 660, bottom: 690 } as DOMRect
    return { top: 0, bottom: 0 } as DOMRect
  })
}

describe('Dropdown', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('opens upward when the scroll area has no room below it', () => {
    nearScrollerBottom()
    render(
      <div data-testid="scroller" style={{ overflowY: 'auto' }}>
        <Dropdown options={options} value="relative" label="Timestamps" />
      </div>,
    )

    fireEvent.click(screen.getByRole('button', { name: 'Timestamps' }))

    expect(screen.getByRole('listbox')).toHaveClass('bottom-full')
  })

  it('moves the highlight without scrolling the page', () => {
    const original = Element.prototype.scrollIntoView
    const scrollIntoView = vi.fn()
    Element.prototype.scrollIntoView = scrollIntoView
    render(<Dropdown options={options} value="relative" label="Timestamps" />)

    fireEvent.click(screen.getByRole('button', { name: 'Timestamps' }))
    fireEvent.keyDown(document, { key: 'ArrowDown' })

    expect(scrollIntoView).not.toHaveBeenCalled()
    Element.prototype.scrollIntoView = original
  })

  it('lets options be wider than the trigger', () => {
    render(<Dropdown options={options} value="relative" label="Timestamps" />)

    fireEvent.click(screen.getByRole('button', { name: 'Timestamps' }))

    expect(screen.getByRole('listbox')).toHaveClass('min-w-full', 'w-max')
  })
})
