import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import type { ConsumerInfo } from '@/types/nats'
import { ConsumerConfirmDialog, type ConsumerConfirmAction } from './ConsumerConfirmDialog'

const consumer = { name: 'orders-worker' } as ConsumerInfo

function pauseAction(pauseMinutes: number | undefined): ConsumerConfirmAction {
  return { type: 'pause', consumer, pauseMinutes }
}

describe('ConsumerConfirmDialog — pause duration', () => {
  it('disables confirm and shows an error for a negative duration', () => {
    render(
      <ConsumerConfirmDialog action={pauseAction(-5)} onChange={vi.fn()} onCancel={vi.fn()} onConfirm={vi.fn()} />,
    )

    expect(screen.getByRole('button', { name: 'Pause' })).toBeDisabled()
    expect(screen.getByText(/Enter 1–525600 minutes/)).toBeInTheDocument()
  })

  it('disables confirm for zero instead of silently substituting a default', () => {
    render(
      <ConsumerConfirmDialog action={pauseAction(0)} onChange={vi.fn()} onCancel={vi.fn()} onConfirm={vi.fn()} />,
    )

    expect(screen.getByLabelText('Pause Duration (minutes)')).toHaveValue(0)
    expect(screen.getByRole('button', { name: 'Pause' })).toBeDisabled()
  })

  it('disables confirm for a duration far beyond a reasonable pause window', () => {
    render(
      <ConsumerConfirmDialog action={pauseAction(99999999999)} onChange={vi.fn()} onCancel={vi.fn()} onConfirm={vi.fn()} />,
    )

    expect(screen.getByRole('button', { name: 'Pause' })).toBeDisabled()
  })

  it('enables confirm for a valid duration', () => {
    render(
      <ConsumerConfirmDialog action={pauseAction(5)} onChange={vi.fn()} onCancel={vi.fn()} onConfirm={vi.fn()} />,
    )

    expect(screen.getByRole('button', { name: 'Pause' })).not.toBeDisabled()
  })

  it('reports the typed value as-is, without an || 5 fallback', () => {
    const onChange = vi.fn()
    render(
      <ConsumerConfirmDialog action={pauseAction(5)} onChange={onChange} onCancel={vi.fn()} onConfirm={vi.fn()} />,
    )

    fireEvent.change(screen.getByLabelText('Pause Duration (minutes)'), { target: { value: '0' } })

    expect(onChange).toHaveBeenCalledWith(pauseAction(0))
  })
})

function resetAction(resetSequence: string | undefined, deliverPolicy = 'all'): ConsumerConfirmAction {
  return {
    type: 'reset',
    consumer: { name: 'orders-worker', config: { deliver_policy: deliverPolicy } } as ConsumerInfo,
    resetSequence,
  }
}

describe('ConsumerConfirmDialog — reset', () => {
  it('allows a reset without a sequence', () => {
    render(
      <ConsumerConfirmDialog action={resetAction(undefined)} onChange={vi.fn()} onCancel={vi.fn()} onConfirm={vi.fn()} />,
    )

    expect(screen.getByRole('heading', { name: 'Reset Consumer' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Reset' })).not.toBeDisabled()
  })

  it.each(['0', 'abc', '-3', '1.5'])('rejects %s as a sequence', (value) => {
    render(
      <ConsumerConfirmDialog action={resetAction(value)} onChange={vi.fn()} onCancel={vi.fn()} onConfirm={vi.fn()} />,
    )

    expect(screen.getByRole('button', { name: 'Reset' })).toBeDisabled()
    expect(screen.getByText('Enter a positive stream sequence, or leave empty')).toBeInTheDocument()
  })

  it('accepts a positive sequence', () => {
    render(
      <ConsumerConfirmDialog action={resetAction('42')} onChange={vi.fn()} onCancel={vi.fn()} onConfirm={vi.fn()} />,
    )

    expect(screen.getByRole('button', { name: 'Reset' })).not.toBeDisabled()
  })

  it('reports the typed sequence', () => {
    const onChange = vi.fn()
    render(
      <ConsumerConfirmDialog action={resetAction(undefined)} onChange={onChange} onCancel={vi.fn()} onConfirm={vi.fn()} />,
    )

    fireEvent.change(screen.getByLabelText('Start from stream sequence (optional)'), { target: { value: '42' } })

    expect(onChange).toHaveBeenCalledWith(resetAction('42'))
  })

  it('disables the sequence for a deliver policy the server cannot rewind', () => {
    render(
      <ConsumerConfirmDialog action={resetAction(undefined, 'last')} onChange={vi.fn()} onCancel={vi.fn()} onConfirm={vi.fn()} />,
    )

    expect(screen.getByLabelText('Start from stream sequence (optional)')).toBeDisabled()
    expect(
      screen.getByText('Only consumers delivering all messages, or from a start sequence or time, can reset to a sequence.'),
    ).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Reset' })).not.toBeDisabled()
  })
})
