import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { ConnectionBadges } from './ConnectionBadges'

describe('ConnectionBadges', () => {
  it('shows the label in its color and marks a read-only connection', () => {
    render(<ConnectionBadges policy={{ readOnly: true, label: { text: 'PROD', color: 'red' } }} />)

    expect(screen.getByText('PROD')).toHaveAttribute('data-color', 'red')
    expect(screen.getByText('Read-only')).toBeInTheDocument()
  })

  it('renders nothing for a writable connection without a label', () => {
    const { container } = render(<ConnectionBadges policy={{ readOnly: false, label: null }} />)
    expect(container).toBeEmptyDOMElement()
  })
})
