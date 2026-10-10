import { beforeEach, describe, expect, it, vi } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'

const create = vi.hoisted(() => ({ mutateAsync: vi.fn(), isPending: false }))

vi.mock('react-router-dom', async (importOriginal) => ({
  ...(await importOriginal<typeof import('react-router-dom')>()),
  useOutletContext: () => ({ connectionId: 'c1', currentConnection: { name: 'local', urls: ['nats://a'] } }),
}))
vi.mock('@/contexts/streams', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/contexts/streams')>()),
  useCreateStream: () => create,
}))
vi.mock('@/contexts/connection', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/contexts/connection')>()),
  useActiveConnection: () => ({ connectionId: 'c1' }),
  useServerCapabilities: () => ({ unsupportedReason: () => undefined }),
}))

import CreateStreamPage from './CreateStreamPage'

function renderPage() {
  render(
    <MemoryRouter>
      <CreateStreamPage />
    </MemoryRouter>,
  )
}

const createButton = () => screen.getByRole('button', { name: 'Create Stream' })

describe('CreateStreamPage', () => {
  beforeEach(() => {
    create.mutateAsync.mockReset()
    create.mutateAsync.mockResolvedValue({ name: 'X' })
  })

  it('says why Create Stream is disabled, and what to do next as the form fills in', () => {
    renderPage()

    expect(createButton()).toBeDisabled()
    expect(screen.getByTestId('create-blocked-reason')).toHaveTextContent('Enter a stream name.')

    fireEvent.change(screen.getByPlaceholderText('my-stream'), { target: { value: 'ORDERS' } })
    expect(screen.getByTestId('create-blocked-reason')).toHaveTextContent('Add at least one subject')

    fireEvent.change(screen.getByPlaceholderText('orders.>'), { target: { value: 'orders.>' } })
    expect(screen.queryByTestId('create-blocked-reason')).toBeNull()
    expect(createButton()).toBeEnabled()
  })

  it('refuses a stream name with a space and says so inline', () => {
    renderPage()

    fireEvent.change(screen.getByPlaceholderText('my-stream'), { target: { value: 'A3 bad name' } })
    fireEvent.change(screen.getByPlaceholderText('orders.>'), { target: { value: 'a3.>' } })

    expect(createButton()).toBeDisabled()
    expect(screen.getAllByText(/cannot contain spaces/).length).toBeGreaterThan(0)
  })

  it('refuses a subject with an empty part', () => {
    renderPage()

    fireEvent.change(screen.getByPlaceholderText('my-stream'), { target: { value: 'A3' } })
    fireEvent.change(screen.getByPlaceholderText('orders.>'), { target: { value: 'a3..x' } })

    expect(createButton()).toBeDisabled()
    expect(screen.getAllByText(/empty parts/).length).toBeGreaterThan(0)
  })

  it('blocks creation while a duration cannot be read', () => {
    renderPage()
    fireEvent.change(screen.getByPlaceholderText('my-stream'), { target: { value: 'ORDERS' } })
    fireEvent.change(screen.getByPlaceholderText('orders.>'), { target: { value: 'orders.>' } })

    fireEvent.change(screen.getByLabelText('Max Age'), { target: { value: 'soon' } })

    expect(createButton()).toBeDisabled()
    expect(screen.getByTestId('create-blocked-reason')).toHaveTextContent(/Max Age/)
  })

  it('sends a typed duration as nanoseconds', async () => {
    renderPage()
    fireEvent.change(screen.getByPlaceholderText('my-stream'), { target: { value: 'ORDERS' } })
    fireEvent.change(screen.getByPlaceholderText('orders.>'), { target: { value: 'orders.>' } })
    fireEvent.change(screen.getByLabelText('Max Age'), { target: { value: '1h' } })

    fireEvent.click(createButton())

    await waitFor(() => expect(create.mutateAsync).toHaveBeenCalled())
    expect(create.mutateAsync.mock.calls[0][0]).toMatchObject({ name: 'ORDERS', max_age: 3_600_000_000_000 })
  })
})
