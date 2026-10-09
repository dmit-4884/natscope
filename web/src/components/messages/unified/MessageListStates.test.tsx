import { afterEach, describe, expect, it, vi } from 'vitest'
import { act, render, screen } from '@testing-library/react'
import { MessagesLoading, RealtimeStatusBar } from './MessageListStates'

describe('MessagesLoading', () => {
  it('stays out of sight for a quick answer', () => {
    const { container } = render(<MessagesLoading />)
    expect(container.firstElementChild).toHaveClass('reveal-after-delay')
  })
})

describe('RealtimeStatusBar', () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  it('takes no room for a status that clears within a moment', () => {
    vi.useFakeTimers()
    render(<RealtimeStatusBar status="connecting" />)
    expect(screen.queryByText('Connecting...')).not.toBeInTheDocument()

    act(() => vi.advanceTimersByTime(250))
    expect(screen.getByText('Connecting...')).toBeInTheDocument()
  })
})
