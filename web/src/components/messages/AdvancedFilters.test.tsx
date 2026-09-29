import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, screen, fireEvent, cleanup } from '@testing-library/react'
import AdvancedFilters, { type FilterValues } from './AdvancedFilters'

const baseFilters: FilterValues = { subject: '', startSequence: null, startDate: null, contentFilter: '' }

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

  it('fills a time-only entry with the local date, not the UTC date', () => {
    expect(new Date('2026-06-12T23:30:00Z').getDate()).toBe(13)
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-06-12T23:30:00Z'))

    render(<AdvancedFilters filters={baseFilters} onFiltersChange={vi.fn()} onClose={vi.fn()} />)

    fireEvent.change(screen.getByLabelText('Start time'), { target: { value: '02:30' } })

    expect(screen.getByLabelText('Start date')).toHaveValue('2026-06-13')
  })
})
