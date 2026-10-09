import { useState } from 'react'
import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@/test/utils'
import { SubjectAutocomplete } from './SubjectAutocomplete'

function Harness({ options = ['orders.>', 'audit.*'] }: { options?: string[] }) {
  const [value, setValue] = useState('')
  return <SubjectAutocomplete value={value} onChange={setValue} options={options} inputId="subject" />
}

describe('SubjectAutocomplete', () => {
  it('opens upward when the scroll area has no room below the field', () => {
    vi.spyOn(HTMLElement.prototype, 'scrollHeight', 'get').mockReturnValue(66)
    vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockImplementation(function (this: HTMLElement) {
      if (this.dataset.testid === 'scroller') return { top: 0, bottom: 705 } as DOMRect
      if (this.tagName === 'INPUT') return { top: 660, bottom: 690 } as DOMRect
      return { top: 0, bottom: 0 } as DOMRect
    })
    render(
      <div data-testid="scroller" style={{ overflowY: 'auto' }}>
        <Harness />
      </div>,
    )

    fireEvent.focus(screen.getByRole('combobox'))

    expect(screen.getByRole('listbox')).toHaveClass('bottom-full')
    vi.restoreAllMocks()
  })

  it('is a combobox whose suggestions are a listbox', () => {
    render(<Harness />)
    const input = screen.getByRole('combobox')
    fireEvent.focus(input)

    expect(input).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByRole('listbox')).toBeInTheDocument()
    expect(screen.getAllByRole('option').map((o) => o.textContent)).toEqual(['orders.>', 'audit.*'])
  })

  it('closes when the field loses focus', () => {
    render(<Harness />)
    const input = screen.getByRole('combobox')
    fireEvent.focus(input)
    fireEvent.blur(input)

    expect(screen.queryByRole('listbox')).not.toBeInTheDocument()
    expect(input).toHaveAttribute('aria-expanded', 'false')
  })

  it('marks the highlighted option and picks it with Enter', () => {
    render(<Harness />)
    const input = screen.getByRole('combobox')
    fireEvent.focus(input)
    fireEvent.keyDown(input, { key: 'ArrowDown' })

    const option = screen.getByRole('option', { name: 'orders.>' })
    expect(option).toHaveAttribute('aria-selected', 'true')
    expect(input).toHaveAttribute('aria-activedescendant', option.id)

    fireEvent.keyDown(input, { key: 'Enter' })
    expect(input).toHaveValue('orders.>')
  })

  it('keeps the typed subject on Enter until a suggestion is chosen with the arrows', () => {
    render(<Harness options={['orders.created.v2']} />)
    const input = screen.getByRole('combobox')
    fireEvent.change(input, { target: { value: 'orders.created' } })

    expect(screen.getByRole('option', { name: 'orders.created.v2' })).toHaveAttribute('aria-selected', 'false')
    fireEvent.keyDown(input, { key: 'Enter' })
    expect(input).toHaveValue('orders.created')
  })
})
