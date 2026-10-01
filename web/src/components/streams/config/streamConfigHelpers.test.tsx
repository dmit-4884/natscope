import { describe, it, expect } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import type { ReplicaInfo } from '@/types/nats'
import { ReplicaBadge } from './streamConfigHelpers'

const replica = (over: Partial<ReplicaInfo> = {}): ReplicaInfo => ({
  name: 'hub-1',
  current: true,
  offline: false,
  active: 170_000_000,
  lag: 0,
  ...over,
})

describe('ReplicaBadge', () => {
  it('shows only the name for a replica in sync', () => {
    render(<ReplicaBadge replica={replica()} />)

    expect(screen.getByTestId('replica-hub-1')).toHaveTextContent(/^hub-1$/)
  })

  it('names the state of an offline replica and says when it was last seen', async () => {
    render(<ReplicaBadge replica={replica({ name: 'hub-3', current: false, offline: true, active: 34_710_171_151_042 })} />)

    expect(screen.getByTestId('replica-hub-3')).toHaveTextContent('hub-3offline')
    fireEvent.mouseEnter(screen.getByTestId('replica-hub-3'))
    expect(await screen.findByRole('tooltip')).toHaveTextContent('Offline, last seen 9h 38m ago')
  })

  it('shows how far a lagging replica is behind', () => {
    render(<ReplicaBadge replica={replica({ name: 'hub-2', current: false, lag: 1234 })} />)

    expect(screen.getByTestId('replica-hub-2')).toHaveTextContent('hub-21.2K behind')
  })
})
