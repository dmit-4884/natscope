import { describe, it, expect, vi, beforeEach } from 'vitest'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
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

describe('ConnectionEditPage — Test of a saved connection', () => {
  const saved: connectionsApi.SavedConnection = {
    id: 'conn-1',
    name: 'prod',
    urls: ['nats://saved:4222'],
    readOnly: false,
    createdAt: 0,
    updatedAt: 0,
  }

  beforeEach(() => {
    vi.clearAllMocks()
    api.getConnections.mockResolvedValue([saved])
    api.testConnection.mockResolvedValue({ success: true, checks: [] } as unknown as Awaited<ReturnType<typeof api.testConnection>>)
  })

  function renderEdit() {
    render(
      <MemoryRouter initialEntries={['/settings/connections/conn-1/edit']}>
        <Routes>
          <Route path="/settings/connections/:id/edit" element={<ConnectionEditPage mode="edit" />} />
        </Routes>
      </MemoryRouter>,
    )
  }

  it('says the test used the saved servers when they were edited', async () => {
    renderEdit()
    const url = await screen.findByDisplayValue('nats://saved:4222')

    fireEvent.change(url, { target: { value: 'nats://edited:4222' } })
    fireEvent.click(screen.getByRole('button', { name: 'Test' }))

    expect(await screen.findByText(/used the saved servers, credentials and TLS settings/)).toBeInTheDocument()
  })

  it('says nothing of the kind when only other settings changed', async () => {
    renderEdit()
    await screen.findByDisplayValue('nats://saved:4222')

    fireEvent.change(screen.getByLabelText('JetStream domain'), { target: { value: 'hub' } })
    fireEvent.click(screen.getByRole('button', { name: 'Test' }))

    expect(await screen.findByText('Success')).toBeInTheDocument()
    expect(screen.queryByText(/used the saved servers/)).not.toBeInTheDocument()
  })
})
