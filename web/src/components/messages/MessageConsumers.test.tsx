import { describe, it, expect, vi, beforeEach } from 'vitest'
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom'
import { render, screen, fireEvent, waitFor } from '@/test/utils'
import { listConsumers } from '@/api/management'
import { AccessDeniedError } from '@/shared/domain/access'
import type { ConsumerInfo } from '@/types/nats'
import { MessageConsumers } from './MessageConsumers'

vi.mock('@/api/management', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/api/management')>()),
  listConsumers: vi.fn(),
}))

const mockedList = vi.mocked(listConsumers)

function consumer(name: string, overrides: Partial<ConsumerInfo> = {}): ConsumerInfo {
  return {
    name,
    stream_name: 'ORDERS',
    num_pending: 0,
    num_ack_pending: 0,
    config: { ack_policy: 'explicit', deliver_policy: 'all' },
    delivered: { consumer_seq: 8, stream_seq: 20 },
    ack_floor: { consumer_seq: 5, stream_seq: 12 },
    ...overrides,
  }
}

function Location() {
  const location = useLocation()
  return <p data-testid="location">{location.pathname + location.search}</p>
}

function renderFate(sequence = 15, subject = 'orders.created') {
  return render(
    <MemoryRouter initialEntries={['/streams/ORDERS/messages']}>
      <Routes>
        <Route
          path="/streams/:streamName/messages"
          element={<MessageConsumers connectionId="conn-1" streamName="ORDERS" message={{ sequence, subject, timestamp: Date.now() }} />}
        />
        <Route path="/streams/:streamName/consumers" element={<Location />} />
      </Routes>
    </MemoryRouter>,
  )
}

describe('MessageConsumers', () => {
  beforeEach(() => {
    mockedList.mockReset()
  })

  it('sums up what each consumer did with the message and lists them on demand', async () => {
    mockedList.mockResolvedValue([
      consumer('billing'),
      consumer('audit', { ack_floor: { consumer_seq: 8, stream_seq: 20 } }),
      consumer('mailer', { delivered: { consumer_seq: 2, stream_seq: 9 }, ack_floor: { consumer_seq: 2, stream_seq: 9 } }),
      consumer('shipping', { config: { filter_subject: 'orders.shipped' } }),
    ])
    renderFate()

    const toggle = await screen.findByRole('button', { name: /consumers:/i })
    await waitFor(() => expect(toggle).toHaveTextContent('1 acknowledged · 1 waiting for ack · 1 not delivered yet'))
    expect(toggle).toHaveAttribute('aria-expanded', 'false')

    fireEvent.click(toggle)
    expect(screen.getByText('billing').closest('li')).toHaveTextContent('Waiting for ack')
    expect(screen.getByText('audit').closest('li')).toHaveTextContent('Acknowledged')
    expect(screen.getByText('mailer').closest('li')).toHaveTextContent('Not delivered yet')
    expect(screen.getByText(/not for 1 other consumer: the filter leaves orders\.created out/i)).toBeInTheDocument()

    fireEvent.click(screen.getByRole('link', { name: 'billing' }))
    expect(screen.getByTestId('location')).toHaveTextContent('/streams/ORDERS/consumers?consumer=billing')
  })

  it('says when no consumer reads the stream', async () => {
    mockedList.mockResolvedValue([])
    renderFate()

    expect(await screen.findByText(/no consumer reads this stream/i)).toBeInTheDocument()
  })

  it('names the permission when consumers cannot be listed', async () => {
    mockedList.mockRejectedValue(
      new AccessDeniedError('refused', { status: 'denied', operation: 'publish', subject: '$JS.API.CONSUMER.LIST.ORDERS' }),
    )
    renderFate()

    expect(await screen.findByText(/publish to \$JS\.API\.CONSUMER\.LIST\.ORDERS/)).toBeInTheDocument()
  })
})
