import { afterEach, describe, expect, it, vi } from 'vitest'
import { act, fireEvent, render, screen } from '@/test/utils'
import CompactHeader from './CompactHeader'

const health = vi.hoisted(() => ({ status: 'connecting' }))

vi.mock('@/contexts/connection', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/contexts/connection')>()),
  useConnectionHealth: () => ({ status: health.status, serverVersion: undefined, refetch: vi.fn() }),
  useConnections: () => ({ data: [], refetch: vi.fn(), isLoading: false }),
  useConnectionPolicy: () => ({ readOnly: false, label: null, known: true }),
}))

const connection = { id: 'conn-1', name: 'local', urls: ['nats://127.0.0.1:4222'] }

function renderHeader() {
  return render(
    <CompactHeader
      connectionId="conn-1"
      currentConnection={connection}
      onSwitchConnection={vi.fn()}
      onDisconnect={vi.fn()}
      onOpenConnections={vi.fn()}
    />,
  )
}

describe('CompactHeader', () => {
  afterEach(() => {
    vi.useRealTimers()
    health.status = 'connecting'
  })

  it('says Connecting only when connecting takes a while', () => {
    vi.useFakeTimers()
    renderHeader()
    expect(screen.queryByText('Connecting')).not.toBeInTheDocument()

    act(() => vi.advanceTimersByTime(350))
    expect(screen.getByText('Connecting')).toBeInTheDocument()
  })

  it('announces its connection menu and closes it on Escape', () => {
    health.status = 'connected'
    renderHeader()
    const trigger = screen.getByRole('button', { name: /local/ })
    expect(trigger).toHaveAttribute('aria-haspopup', 'true')

    fireEvent.click(trigger)
    expect(trigger).toHaveAttribute('aria-expanded', 'true')
    fireEvent.keyDown(document, { key: 'Escape' })

    expect(trigger).toHaveAttribute('aria-expanded', 'false')
    expect(screen.queryByText('Manage Connections')).not.toBeInTheDocument()
  })
})
