import { describe, it, expect, vi, beforeEach } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import { render, screen, fireEvent } from '@/test/utils'
import * as connectionsApi from '@/api/connections'
import ConnectionEditPage from './ConnectionEditPage'

vi.mock('@/api/connections', async (importOriginal) => ({
  ...(await importOriginal<typeof connectionsApi>()),
  getConnections: vi.fn(),
  testConnection: vi.fn(),
}))

const api = vi.mocked(connectionsApi)

describe('ConnectionEditPage — Test', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    api.getConnections.mockResolvedValue([])
  })

  it('shows an invalid JetStream target at its field instead of sending the test', async () => {
    render(
      <MemoryRouter>
        <ConnectionEditPage mode="create" />
      </MemoryRouter>,
    )

    fireEvent.change(screen.getByLabelText('Server URL 1'), { target: { value: 'nats://localhost:4222' } })
    fireEvent.change(screen.getByLabelText('JetStream domain'), { target: { value: 'hub' } })
    fireEvent.change(screen.getByLabelText('JetStream API prefix'), { target: { value: 'JS.orders.API' } })
    fireEvent.click(screen.getByRole('button', { name: 'Test' }))

    expect(await screen.findByText('Set a JetStream domain or an API prefix, not both')).toBeInTheDocument()
    expect(api.testConnection).not.toHaveBeenCalled()
  })
})
