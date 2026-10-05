import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import type { ComponentProps } from 'react'
import type { ConsumerInfo } from '@/types/nats'
import { ConsumerView } from './ConsumerView'

const consumer: ConsumerInfo = {
  name: 'worker',
  num_pending: 3,
  num_ack_pending: 0,
  config: { deliver_policy: 'all', ack_policy: 'explicit' },
} as ConsumerInfo

function renderView(overrides: Partial<ComponentProps<typeof ConsumerView>> = {}) {
  const props: ComponentProps<typeof ConsumerView> = {
    consumer,
    streamName: 'ORDERS',
    isFetching: false,
    onRefetch: vi.fn(),
    onEdit: vi.fn(),
    onPause: vi.fn(),
    onResume: vi.fn(),
    onReset: vi.fn(),
    onUnpin: vi.fn(),
    onDelete: vi.fn(),
    isResuming: false,
    isPausing: false,
    isResetting: false,
    isUnpinning: false,
    ...overrides,
  }
  render(<ConsumerView {...props} />)
  return props
}

describe('ConsumerView — reset', () => {
  it('asks to reset when the server supports it', () => {
    const props = renderView()

    fireEvent.click(screen.getByRole('button', { name: 'Reset' }))

    expect(props.onReset).toHaveBeenCalledOnce()
  })

  it('disables reset on servers without consumer reset', () => {
    const props = renderView({ resetUnsupportedReason: 'Requires NATS 2.14+ (connected server is v2.12.3)' })

    const button = screen.getByRole('button', { name: 'Reset' })
    expect(button).toBeDisabled()
    fireEvent.click(button)
    expect(props.onReset).not.toHaveBeenCalled()
  })

  it('disables reset while a reset is in flight', () => {
    renderView({ isResetting: true })

    expect(screen.getByRole('button', { name: 'Reset' })).toBeDisabled()
  })
})

describe('ConsumerView — problems', () => {
  it('explains what holds the consumer back', () => {
    renderView({
      issues: [
        { kind: 'ack_limit', severity: 'error', label: 'Ack limit reached', detail: '1000 messages wait for an ack.' },
        { kind: 'redelivering', severity: 'warning', label: 'Redelivering', detail: '3 messages were delivered again.' },
      ],
    })

    const problems = screen.getByTestId('consumer-problems')
    expect(problems).toHaveTextContent('Ack limit reached')
    expect(problems).toHaveTextContent('1000 messages wait for an ack.')
    expect(problems).toHaveTextContent('3 messages were delivered again.')
  })

  it('stays quiet when nothing is wrong', () => {
    renderView({ issues: [] })

    expect(screen.queryByTestId('consumer-problems')).not.toBeInTheDocument()
  })

  it('shows where the consumer is when given', () => {
    renderView({ position: <p>position card</p> })

    expect(screen.getByText('position card')).toBeInTheDocument()
  })
})
