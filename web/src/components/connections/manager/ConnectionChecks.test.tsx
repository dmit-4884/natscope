import { describe, it, expect } from 'vitest'
import { render, screen, within } from '@testing-library/react'
import { ConnectionChecks } from './ConnectionChecks'

describe('ConnectionChecks', () => {
  it('lists every step with its outcome and what to do about a failure', () => {
    render(
      <ConnectionChecks
        checks={[
          { step: 'dns', status: 'ok', detail: 'nats.example resolves to 10.0.0.4', hint: '', durationMs: 2 },
          { step: 'tcp', status: 'failed', detail: 'Cannot connect to nats.example:4222', hint: 'Is the NATS server running?', durationMs: 5 },
          { step: 'tls', status: 'warning', detail: 'certificate expires in 3 days', hint: 'Renew the server certificate.', durationMs: 1 },
          { step: 'jetstream', status: 'skipped', detail: 'Not reached', hint: '', durationMs: 0 },
        ]}
      />,
    )

    const items = screen.getAllByRole('listitem')
    expect(items).toHaveLength(4)
    expect(within(items[0]).getByText('DNS')).toBeInTheDocument()
    expect(items[0]).toHaveAttribute('data-status', 'ok')
    expect(within(items[1]).getByText('TCP')).toBeInTheDocument()
    expect(within(items[1]).getByText('Is the NATS server running?')).toBeInTheDocument()
    expect(items[1]).toHaveAttribute('data-status', 'failed')
    expect(within(items[2]).getByText('Renew the server certificate.')).toBeInTheDocument()
    expect(within(items[3]).getByText('JetStream')).toBeInTheDocument()
    expect(within(items[3]).getByText('Not reached')).toBeInTheDocument()
  })
})
