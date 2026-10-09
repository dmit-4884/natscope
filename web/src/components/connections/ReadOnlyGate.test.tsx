import { describe, it, expect, vi } from 'vitest'
import type { ReactElement } from 'react'
import { MemoryRouter } from 'react-router-dom'
import { render as renderWithClient, screen } from '@/test/utils'
import { ReadOnlyGate } from './ReadOnlyGate'

const policy = vi.hoisted(() => ({ readOnly: false, known: true }))

vi.mock('@/contexts/connection', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/contexts/connection')>()),
  useConnectionPolicy: () => ({ readOnly: policy.readOnly, label: null, known: policy.known }),
  useActiveConnection: () => ({ connectionId: 'prod-1', connection: null, isLoading: false, isConnected: true }),
}))

const render = (ui: ReactElement) => renderWithClient(<MemoryRouter>{ui}</MemoryRouter>)

describe('ReadOnlyGate', () => {
  it('renders the page on a writable connection', () => {
    policy.readOnly = false
    render(<ReadOnlyGate><p>publish form</p></ReadOnlyGate>)
    expect(screen.getByText('publish form')).toBeInTheDocument()
  })

  it('shows nothing while the connection policy is still unknown', () => {
    policy.readOnly = true
    policy.known = false
    render(<ReadOnlyGate><p>publish form</p></ReadOnlyGate>)

    expect(screen.queryByText('publish form')).not.toBeInTheDocument()
    expect(screen.queryByText('This connection is read-only')).not.toBeInTheDocument()
    policy.known = true
  })

  it('explains a read-only connection and links to its settings', () => {
    policy.readOnly = true
    render(<ReadOnlyGate><p>publish form</p></ReadOnlyGate>)

    expect(screen.queryByText('publish form')).not.toBeInTheDocument()
    expect(screen.getByText('This connection is read-only')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Connection settings' })).toHaveAttribute('href', '/settings/connections/prod-1/edit')
  })
})
