import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import type { ConsumerInfo } from '@/types/nats'
import { ConsumerPriority } from './ConsumerPriority'

function consumer(over: Partial<ConsumerInfo> = {}, config: ConsumerInfo['config'] = {}): ConsumerInfo {
  return { name: 'worker', num_pending: 0, num_ack_pending: 0, config, ...over }
}

describe('ConsumerPriority', () => {
  it('renders nothing for a consumer without a priority policy', () => {
    const { container } = render(
      <ConsumerPriority consumer={consumer()} onUnpin={vi.fn()} isUnpinning={false} />,
    )
    expect(container).toBeEmptyDOMElement()
  })

  it('shows the policy, groups and pinned ttl', () => {
    render(
      <ConsumerPriority
        consumer={consumer({}, { priority_policy: 'pinned_client', priority_groups: ['jobs', 'bulk'], priority_timeout: 120_000_000_000 })}
        onUnpin={vi.fn()}
        isUnpinning={false}
      />,
    )

    expect(screen.getByText('Pinned client')).toBeInTheDocument()
    expect(screen.getByText('jobs')).toBeInTheDocument()
    expect(screen.getByText('bulk')).toBeInTheDocument()
    expect(screen.getByText('2m')).toBeInTheDocument()
  })

  it('unpins the pinned client of a group', () => {
    const onUnpin = vi.fn()
    render(
      <ConsumerPriority
        consumer={consumer(
          { priority_groups: [{ group: 'jobs', pinned_client_id: 'pin-1', pinned_ts: 1_790_000_000_000 }] },
          { priority_policy: 'pinned_client', priority_groups: ['jobs'] },
        )}
        onUnpin={onUnpin}
        isUnpinning={false}
      />,
    )

    expect(screen.getByText('pin-1')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Unpin jobs' }))
    expect(onUnpin).toHaveBeenCalledWith('jobs')
  })

  it('offers no unpin when no client is pinned', () => {
    render(
      <ConsumerPriority
        consumer={consumer({ priority_groups: [{ group: 'jobs' }] }, { priority_policy: 'pinned_client', priority_groups: ['jobs'] })}
        onUnpin={vi.fn()}
        isUnpinning={false}
      />,
    )

    expect(screen.getByText('No client pinned')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Unpin jobs' })).not.toBeInTheDocument()
  })

  it('disables unpin on servers without priority groups', () => {
    render(
      <ConsumerPriority
        consumer={consumer(
          { priority_groups: [{ group: 'jobs', pinned_client_id: 'pin-1' }] },
          { priority_policy: 'pinned_client', priority_groups: ['jobs'] },
        )}
        onUnpin={vi.fn()}
        isUnpinning={false}
        unpinUnsupportedReason="Requires NATS 2.11+"
      />,
    )

    expect(screen.getByRole('button', { name: 'Unpin jobs' })).toBeDisabled()
  })
})
