import { describe, it, expect, vi, beforeEach } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import { ConnectError, Code } from '@connectrpc/connect'
import { render, screen, fireEvent, waitFor } from '@/test/utils'
import { toast } from '@/utils/toast'
import * as connectionsApi from '@/api/connections'
import ConnectionSelector from './ConnectionSelector'

vi.mock('@/utils/toast', () => ({
  toast: { success: vi.fn(), error: vi.fn(), warning: vi.fn(), info: vi.fn() },
}))

vi.mock('@/api/connections', async (importOriginal) => ({
  ...(await importOriginal<typeof connectionsApi>()),
  getConnections: vi.fn(),
  createConnection: vi.fn(),
  testConnection: vi.fn(),
}))

const api = vi.mocked(connectionsApi)

function openNewForm() {
  render(
    <MemoryRouter>
      <ConnectionSelector />
    </MemoryRouter>,
  )
  fireEvent.click(screen.getByRole('button', { name: /New Connection/i }))
}

describe('ConnectionSelector', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    api.getConnections.mockResolvedValue([])
  })

  it('reports a failed create inline only — the global mutation handler owns the toast', async () => {
    api.createConnection.mockRejectedValue(
      new ConnectError('connection name is required', Code.InvalidArgument),
    )
    openNewForm()

    fireEvent.change(screen.getByLabelText('Server URL 1'), {
      target: { value: 'nats://localhost:4222' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Connect' }))

    expect(await screen.findByText('connection name is required')).toBeInTheDocument()
    await waitFor(() => expect(api.createConnection).toHaveBeenCalledTimes(1))
    expect(toast.error).not.toHaveBeenCalled()
  })

  it('reports a failed test inline only', async () => {
    api.testConnection.mockRejectedValue(new ConnectError('boom', Code.Unavailable))
    openNewForm()

    fireEvent.change(screen.getByLabelText('Server URL 1'), {
      target: { value: 'nats://localhost:4222' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Test' }))

    expect(await screen.findByText('boom')).toBeInTheDocument()
    expect(toast.error).not.toHaveBeenCalled()
  })

  it('blocks Test on a malformed server URL instead of dialing it', async () => {
    openNewForm()

    fireEvent.change(screen.getByLabelText('Server URL 1'), { target: { value: 'not-a-url' } })
    fireEvent.click(screen.getByRole('button', { name: 'Test' }))

    expect(await screen.findByText(/nats:\/\//)).toBeInTheDocument()
    expect(api.testConnection).not.toHaveBeenCalled()
  })

  it('flags a whitespace-only name inline instead of letting the server reject it', async () => {
    openNewForm()

    fireEvent.change(screen.getByLabelText('Name'), { target: { value: '   ' } })
    fireEvent.change(screen.getByLabelText('Server URL 1'), {
      target: { value: 'nats://localhost:4222' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Connect' }))

    expect(await screen.findByText('Name is required')).toBeInTheDocument()
    expect(api.createConnection).not.toHaveBeenCalled()
  })

  it('strips the [code] prefix from a rejected probe result', async () => {
    api.testConnection.mockResolvedValue({
      success: false,
      error: '[unavailable] nats: no servers available for connection',
    } as unknown as connectionsApi.TestConnectionResponse)
    openNewForm()

    fireEvent.change(screen.getByLabelText('Server URL 1'), {
      target: { value: 'nats://localhost:4222' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Test' }))

    await waitFor(() =>
      expect(toast.error).toHaveBeenCalledWith('nats: no servers available for connection'),
    )
    expect(toast.error).toHaveBeenCalledTimes(1)
  })
})
