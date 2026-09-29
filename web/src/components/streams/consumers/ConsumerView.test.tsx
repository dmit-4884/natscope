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
