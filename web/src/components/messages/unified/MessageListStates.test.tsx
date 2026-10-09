import { describe, expect, it } from 'vitest'
import { render } from '@testing-library/react'
import { MessagesLoading } from './MessageListStates'

describe('MessagesLoading', () => {
  it('stays out of sight for a quick answer', () => {
    const { container } = render(<MessagesLoading />)
    expect(container.firstElementChild).toHaveClass('reveal-after-delay')
  })
})
