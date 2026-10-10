import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { StreamConfirmDialog } from './StreamConfirmDialog'

describe('StreamConfirmDialog', () => {
  it.each([
    ['seal', 'Seal stream'],
    ['delete', 'Delete stream'],
    ['purge', 'Purge stream'],
  ] as const)('labels the %s confirm button so it differs from the page action', (type, label) => {
    render(<StreamConfirmDialog type={type} streamName="ORDERS" onCancel={vi.fn()} onConfirm={vi.fn()} />)
    expect(screen.getByRole('button', { name: label })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: new RegExp(`^${type}$`, 'i') })).not.toBeInTheDocument()
  })
})
