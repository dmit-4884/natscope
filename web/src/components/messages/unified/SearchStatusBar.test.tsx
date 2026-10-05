import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import type { Message } from '@/types/nats'
import type { MessageSearch } from './useMessageSearch'
import { SearchStatusBar } from './SearchStatusBar'

const found = (n: number) => Array.from({ length: n }, (_, i) => ({ sequence: i + 1 }) as Message)

function search(over: Partial<MessageSearch>): MessageSearch {
  return {
    status: 'running',
    messages: [],
    progress: null,
    done: null,
    error: null,
    scanned: 0,
    range: { first: 1, last: 1_000_000 },
    canContinue: false,
    more: vi.fn(),
    stop: vi.fn(),
    restart: vi.fn(),
    ...over,
  }
}

describe('SearchStatusBar', () => {
  it('shows how much a running search read and found, and stops it', () => {
    const s = search({
      scanned: 120_000,
      messages: found(3),
      progress: { scanned: 120_000, matched: 3, current_seq: 870_000, range_first: 1, range_last: 1_000_000, resume_seq: 880_000 },
    })
    render(<SearchStatusBar search={s} direction="backward" />)

    expect(screen.getByTestId('search-status')).toHaveTextContent('Searching… 120,000 messages read · 3 found')
    expect(screen.getByRole('progressbar')).toHaveAttribute('aria-valuenow', '12')
    fireEvent.click(screen.getByRole('button', { name: 'Stop' }))
    expect(s.stop).toHaveBeenCalled()
  })

  it('says the whole range was searched', () => {
    render(
      <SearchStatusBar
        search={search({
          status: 'done',
          scanned: 1_000_000,
          messages: found(2),
          done: { scanned: 1_000_000, matched: 2, reason: 'complete', range_first: 1, range_last: 1_000_000 },
        })}
        direction="backward"
      />,
    )
    expect(screen.getByTestId('search-status')).toHaveTextContent('Searched every message from #1 to #1,000,000: 1,000,000 messages read · 2 found')
    expect(screen.queryByRole('button', { name: 'Search further' })).not.toBeInTheDocument()
  })

  it('explains a budget stop and searches further', () => {
    const s = search({
      status: 'done',
      scanned: 100_000,
      messages: found(1),
      canContinue: true,
      done: { scanned: 100_000, matched: 1, reason: 'scan_limit', range_first: 1, range_last: 1_000_000, next_seq: 900_000 },
    })
    render(<SearchStatusBar search={s} direction="backward" />)

    expect(screen.getByTestId('search-status')).toHaveTextContent(
      '100,000 messages read · 1 found. One search reads up to 100,000 messages; the next one continues from #900,000.',
    )
    fireEvent.click(screen.getByRole('button', { name: 'Search further' }))
    expect(s.more).toHaveBeenCalled()
  })

  it('names the other budgets', () => {
    const { rerender } = render(
      <SearchStatusBar
        search={search({ status: 'done', canContinue: true, done: { scanned: 5, matched: 0, reason: 'time_limit', range_first: 1, range_last: 9, next_seq: 6 } })}
        direction="forward"
      />,
    )
    expect(screen.getByTestId('search-status')).toHaveTextContent('One search runs up to 20 seconds')

    rerender(
      <SearchStatusBar
        search={search({ status: 'done', canContinue: true, done: { scanned: 5, matched: 500, reason: 'match_limit', range_first: 1, range_last: 9, next_seq: 6 } })}
        direction="forward"
      />,
    )
    expect(screen.getByTestId('search-status')).toHaveTextContent('One search returns up to 500 matches')
  })

  it('says when the range holds nothing to search', () => {
    render(<SearchStatusBar search={search({ status: 'done', range: null, done: { scanned: 0, matched: 0, reason: 'complete', range_first: 0, range_last: 0 } })} direction="backward" />)
    expect(screen.getByTestId('search-status')).toHaveTextContent('Nothing to search in this range')
  })

  it('announces the state, not every count', () => {
    const { rerender } = render(<SearchStatusBar search={search({ scanned: 10 })} direction="backward" />)
    expect(screen.getByRole('status')).toHaveTextContent(/^Searching$/)
    rerender(<SearchStatusBar search={search({ status: 'done', scanned: 50, done: { scanned: 50, matched: 0, reason: 'complete', range_first: 1, range_last: 9 } })} direction="backward" />)
    expect(screen.getByRole('status')).toHaveTextContent(/^Search finished$/)
  })

  it('reports a stopped search and a failed one', () => {
    const stopped = search({ status: 'stopped', scanned: 4_000, messages: found(1), canContinue: true })
    const { rerender } = render(<SearchStatusBar search={stopped} direction="backward" />)
    expect(screen.getByTestId('search-status')).toHaveTextContent('Stopped after 4,000 messages read · 1 found')
    expect(screen.getByRole('button', { name: 'Search further' })).toBeInTheDocument()

    const failed = search({ status: 'error', error: new Error('invalid regular expression: missing )') })
    rerender(<SearchStatusBar search={failed} direction="backward" />)
    expect(screen.getByTestId('search-status')).toHaveTextContent('Search failed: invalid regular expression: missing )')
    fireEvent.click(screen.getByRole('button', { name: 'Try again' }))
    expect(failed.restart).toHaveBeenCalled()
  })
})
