import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, screen, fireEvent, cleanup } from '@testing-library/react'
import AdvancedFilters, { type FilterValues } from './AdvancedFilters'
import { EMPTY_FILTERS } from './searchQuery'

const baseFilters: FilterValues = { ...EMPTY_FILTERS }

describe('AdvancedFilters', () => {
  afterEach(() => {
    cleanup()
    vi.useRealTimers()
  })

  it('gives the date and time inputs their own accessible names', () => {
    render(<AdvancedFilters filters={baseFilters} onFiltersChange={vi.fn()} onClose={vi.fn()} />)

    expect(screen.getByLabelText('Start date')).toHaveAttribute('type', 'date')
    expect(screen.getByLabelText('Start time')).toHaveAttribute('type', 'time')
    expect(screen.getByLabelText('Subject')).toBeInTheDocument()
    expect(screen.getByLabelText('Payload Search')).toBeInTheDocument()
    expect(screen.getByLabelText('Start Sequence')).toBeInTheDocument()
  })

  it('fits below its button so Apply stays on screen', () => {
    vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockReturnValue({ top: 231 } as DOMRect)
    vi.spyOn(window, 'innerHeight', 'get').mockReturnValue(900)

    render(<AdvancedFilters filters={baseFilters} onFiltersChange={vi.fn()} onClose={vi.fn()} />)

    expect(screen.getByRole('dialog', { name: 'Filters' }).style.maxHeight).toBe('661px')
    vi.restoreAllMocks()
  })

  it('closes on Escape', () => {
    const onClose = vi.fn()
    render(<AdvancedFilters filters={baseFilters} onFiltersChange={vi.fn()} onClose={onClose} />)

    fireEvent.keyDown(screen.getByLabelText('Payload Search'), { key: 'Escape' })

    expect(onClose).toHaveBeenCalled()
  })

  it('applies on Enter in a field', () => {
    const onFiltersChange = vi.fn()
    render(<AdvancedFilters filters={baseFilters} onFiltersChange={onFiltersChange} onClose={vi.fn()} />)
    const payload = screen.getByLabelText('Payload Search')
    fireEvent.change(payload, { target: { value: 'ord-' } })

    fireEvent.submit(payload)

    expect(onFiltersChange).toHaveBeenCalledWith(expect.objectContaining({ contentFilter: 'ord-' }))
  })

  it('fills a time-only entry with the local date, not the UTC date', () => {
    expect(new Date('2026-06-12T23:30:00Z').getDate()).toBe(13)
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-06-12T23:30:00Z'))

    render(<AdvancedFilters filters={baseFilters} onFiltersChange={vi.fn()} onClose={vi.fn()} />)

    fireEvent.change(screen.getByLabelText('Start time'), { target: { value: '02:30' } })

    expect(screen.getByLabelText('Start date')).toHaveValue('2026-06-13')
  })
})
