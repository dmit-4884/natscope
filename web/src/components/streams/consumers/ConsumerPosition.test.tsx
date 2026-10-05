import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@/test/utils'
import { getNextMessage } from '@/api/messages'
import type { ConsumerInfo, Message } from '@/types/nats'
import { ConsumerPosition } from './ConsumerPosition'

vi.mock('@/api/messages', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/api/messages')>()),
  getNextMessage: vi.fn(),
}))

const mockedNext = vi.mocked(getNextMessage)

function message(sequence: number, subject = 'orders.created'): Message {
  return { sequence, subject, timestamp: Date.UTC(2026, 9, 5, 12, 0, 0), data_base64: 'e30=', data_size: 2, content_type: 'json' }
}

function consumer(overrides: Partial<ConsumerInfo> = {}): ConsumerInfo {
  return {
    name: 'billing',
    stream_name: 'ORDERS',
    num_pending: 3,
    num_ack_pending: 2,
    config: { ack_policy: 'explicit', filter_subjects: ['orders.created', 'orders.paid'] },
    delivered: { consumer_seq: 9, stream_seq: 20 },
    ack_floor: { consumer_seq: 7, stream_seq: 12 },
    ...overrides,
  }
}

describe('ConsumerPosition', () => {
  beforeEach(() => {
    mockedNext.mockReset()
  })

  it('finds the oldest unacknowledged and the next message to deliver through the consumer filters', async () => {
    mockedNext.mockImplementation(async (_conn, _stream, startSeq) => (startSeq === 13 ? message(14) : message(22, 'orders.paid')))
    const onOpen = vi.fn()
    render(<ConsumerPosition connectionId="conn-1" streamName="ORDERS" consumer={consumer()} onOpenMessage={onOpen} />)

    expect(await screen.findByText('#14')).toBeInTheDocument()
    expect(await screen.findByText('#22')).toBeInTheDocument()
    expect(mockedNext).toHaveBeenCalledWith('conn-1', 'ORDERS', 13, ['orders.created', 'orders.paid'], expect.anything())
    expect(mockedNext).toHaveBeenCalledWith('conn-1', 'ORDERS', 21, ['orders.created', 'orders.paid'], expect.anything())
    expect(screen.getByTestId('consumer-floor')).toHaveTextContent('Delivered up to #20 · acknowledged up to #12')

    fireEvent.click(screen.getByRole('button', { name: 'Open message #22' }))
    expect(onOpen).toHaveBeenCalledWith(expect.objectContaining({ sequence: 22 }))
  })

  it('says when nothing waits, without asking the server', async () => {
    render(
      <ConsumerPosition
        connectionId="conn-1"
        streamName="ORDERS"
        consumer={consumer({ num_pending: 0, num_ack_pending: 0 })}
        onOpenMessage={vi.fn()}
      />,
    )

    expect(screen.getByText('Nothing is waiting for an ack.')).toBeInTheDocument()
    expect(screen.getByText('Nothing left to deliver.')).toBeInTheDocument()
    expect(mockedNext).not.toHaveBeenCalled()
  })

  it('explains a message that is gone from the stream', async () => {
    mockedNext.mockResolvedValue(null)
    render(<ConsumerPosition connectionId="conn-1" streamName="ORDERS" consumer={consumer({ num_ack_pending: 0 })} onOpenMessage={vi.fn()} />)

    await waitFor(() => expect(screen.getByText(/no longer in the stream/i)).toBeInTheDocument())
  })
})

describe('ConsumerPosition floor line', () => {
  it('says plainly when nothing was delivered or acknowledged yet', () => {
    const fresh = consumer({ num_pending: 0, num_ack_pending: 0, delivered: { consumer_seq: 0, stream_seq: 0 }, ack_floor: { consumer_seq: 0, stream_seq: 0 } })
    render(<ConsumerPosition connectionId="conn-1" streamName="ORDERS" consumer={fresh} onOpenMessage={vi.fn()} />)

    expect(screen.getByTestId('consumer-floor')).toHaveTextContent('Nothing delivered yet')
  })

  it('says when deliveries are not acknowledged yet', () => {
    const unacked = consumer({ num_pending: 0, num_ack_pending: 0, ack_floor: { consumer_seq: 0, stream_seq: 0 } })
    render(<ConsumerPosition connectionId="conn-1" streamName="ORDERS" consumer={unacked} onOpenMessage={vi.fn()} />)

    expect(screen.getByTestId('consumer-floor')).toHaveTextContent('Delivered up to #20 · nothing acknowledged yet')
  })
})
