import { afterEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { SearchableSelect } from './SearchableSelect'

const options = [
  { value: 'a', label: 'alpha' },
  { value: 'b', label: 'beta' },
]

describe('SearchableSelect', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('opens upward when the scroll area has no room below it', () => {
    vi.spyOn(HTMLElement.prototype, 'scrollHeight', 'get').mockReturnValue(120)
    vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockImplementation(function (this: HTMLElement) {
      if (this.dataset.testid === 'scroller') return { top: 0, bottom: 705 } as DOMRect
      if (this.tagName === 'BUTTON') return { top: 660, bottom: 690 } as DOMRect
      return { top: 0, bottom: 0 } as DOMRect
    })
    render(
      <div data-testid="scroller" style={{ overflowY: 'auto' }}>
        <SearchableSelect options={options} label="Stream" />
      </div>,
    )

    fireEvent.click(screen.getByRole('button', { name: 'Stream' }))

    expect(screen.getByRole('listbox').parentElement).toHaveClass('bottom-full')
  })

  it('moves the highlight without scrolling the page', () => {
    const original = Element.prototype.scrollIntoView
    const scrollIntoView = vi.fn()
    Element.prototype.scrollIntoView = scrollIntoView
    render(<SearchableSelect options={options} label="Stream" />)

    fireEvent.click(screen.getByRole('button', { name: 'Stream' }))
    fireEvent.keyDown(screen.getByRole('combobox'), { key: 'ArrowDown' })

    expect(scrollIntoView).not.toHaveBeenCalled()
    Element.prototype.scrollIntoView = original
  })
})
