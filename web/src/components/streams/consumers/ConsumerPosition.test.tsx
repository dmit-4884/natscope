import type { ReactElement } from 'react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import { render as renderWithClient, screen, fireEvent, waitFor } from '@/test/utils'
import { getNextMessage } from '@/api/messages'
import type { ConsumerInfo, Message } from '@/types/nats'
import { ConsumerPosition } from './ConsumerPosition'

vi.mock('@/api/messages', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/api/messages')>()),
  getNextMessage: vi.fn(),
}))

const mockedNext = vi.mocked(getNextMessage)

function render(ui: ReactElement) {
  return renderWithClient(<MemoryRouter>{ui}</MemoryRouter>)
}

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
    expect(screen.getByTestId('consumer-floor')).toHaveTextContent('Delivered up to #20 · done up to #12')

    const open = screen.getByRole('link', { name: 'Open message #22' })
    expect(open).toHaveAttribute('href', '/streams/ORDERS/messages?msg=history-22')
    fireEvent.click(open)
    expect(onOpen).toHaveBeenCalledWith(expect.objectContaining({ sequence: 22 }))
  })

  it('leaves a modified click to the browser so the message opens in a new tab', async () => {
    mockedNext.mockResolvedValue(message(22, 'orders.paid'))
    const onOpen = vi.fn()
    render(<ConsumerPosition connectionId="conn-1" streamName="ORDERS" consumer={consumer({ num_ack_pending: 0 })} onOpenMessage={onOpen} />)

    const open = await screen.findByRole('link', { name: 'Open message #22' })
    const passed = fireEvent.click(open, { metaKey: true })

    expect(passed).toBe(true)
    expect(onOpen).not.toHaveBeenCalled()
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
    const fresh = consumer({ num_pending: 0, num_ack_pending: 0, delivered: { consumer_seq: 0, stream_seq: 40 }, ack_floor: { consumer_seq: 0, stream_seq: 40 } })
    render(<ConsumerPosition connectionId="conn-1" streamName="ORDERS" consumer={fresh} onOpenMessage={vi.fn()} />)

    expect(screen.getByTestId('consumer-floor')).toHaveTextContent('Nothing delivered yet')
  })

  it('says when deliveries are not acknowledged yet', () => {
    const unacked = consumer({ num_pending: 0, num_ack_pending: 0, ack_floor: { consumer_seq: 0, stream_seq: 11 } })
    render(<ConsumerPosition connectionId="conn-1" streamName="ORDERS" consumer={unacked} onOpenMessage={vi.fn()} />)

    expect(screen.getByTestId('consumer-floor')).toHaveTextContent('Delivered up to #20 · nothing done in order yet')
  })
})

describe('ConsumerPosition oldest unacknowledged', () => {
  beforeEach(() => {
    mockedNext.mockReset()
  })

  it('does not pass off a later message when the unacknowledged one left the stream', async () => {
    mockedNext.mockResolvedValue(message(25))
    render(<ConsumerPosition connectionId="conn-1" streamName="ORDERS" consumer={consumer({ num_pending: 0 })} onOpenMessage={vi.fn()} />)

    expect(await screen.findByText(/no longer in the stream/i)).toBeInTheDocument()
    expect(screen.queryByText('#25')).not.toBeInTheDocument()
  })
})

describe('ConsumerPosition lost messages', () => {
  it('says which messages left the stream before the consumer reached them', () => {
    const behind = consumer({ num_pending: 0, num_ack_pending: 0, config: { ack_policy: 'explicit' } })
    render(<ConsumerPosition connectionId="conn-1" streamName="ORDERS" consumer={behind} firstSeq={40} onOpenMessage={vi.fn()} />)

    expect(screen.getByTestId('consumer-lost')).toHaveTextContent('Messages #21–#39 left the stream before this consumer reached them.')
  })

  it('stays quiet when the consumer is ahead of the stream start', () => {
    render(<ConsumerPosition connectionId="conn-1" streamName="ORDERS" consumer={consumer({ num_pending: 0, num_ack_pending: 0 })} firstSeq={5} onOpenMessage={vi.fn()} />)

    expect(screen.queryByTestId('consumer-lost')).not.toBeInTheDocument()
  })
})

describe('ConsumerPosition lost messages for a new consumer', () => {
  it('does not claim losses before the first delivery', () => {
    const fresh = consumer({ num_pending: 0, num_ack_pending: 0, delivered: { consumer_seq: 0, stream_seq: 0 } })
    render(<ConsumerPosition connectionId="conn-1" streamName="ORDERS" consumer={fresh} firstSeq={40} onOpenMessage={vi.fn()} />)

    expect(screen.queryByTestId('consumer-lost')).not.toBeInTheDocument()
  })
})

describe('ConsumerPosition lost messages for a filtered consumer', () => {
  it('makes no claim when the filter skips some of the stream subjects', () => {
    const behind = consumer({ num_pending: 0, num_ack_pending: 0 })
    render(
      <ConsumerPosition connectionId="conn-1" streamName="ORDERS" consumer={behind} firstSeq={40} streamSubjects={['orders.>']} onOpenMessage={vi.fn()} />,
    )

    expect(screen.queryByTestId('consumer-lost')).not.toBeInTheDocument()
  })

  it('names the gap when the filter takes the whole stream', () => {
    const behind = consumer({ num_pending: 0, num_ack_pending: 0, config: { ack_policy: 'explicit', filter_subject: 'orders.>' } })
    render(
      <ConsumerPosition connectionId="conn-1" streamName="ORDERS" consumer={behind} firstSeq={40} streamSubjects={['orders.>']} onOpenMessage={vi.fn()} />,
    )

    expect(screen.getByTestId('consumer-lost')).toHaveTextContent('Messages #21–#39 left the stream')
  })
})

describe('ConsumerPosition with a delivery limit', () => {
  beforeEach(() => {
    mockedNext.mockReset()
  })

  it('admits the oldest unacknowledged may be one that ran out of attempts', async () => {
    mockedNext.mockResolvedValue(message(14))
    const limited = consumer({ config: { ack_policy: 'explicit', max_deliver: 3 } })
    render(<ConsumerPosition connectionId="conn-1" streamName="ORDERS" consumer={limited} onOpenMessage={vi.fn()} />)

    expect(await screen.findByText(/ran out of delivery attempts/)).toBeInTheDocument()
  })

  it('says nothing about attempts without a limit', async () => {
    mockedNext.mockResolvedValue(message(14))
    render(<ConsumerPosition connectionId="conn-1" streamName="ORDERS" consumer={consumer()} onOpenMessage={vi.fn()} />)

    expect(await screen.findByText('#14')).toBeInTheDocument()
    expect(screen.queryByText(/ran out of delivery attempts/)).not.toBeInTheDocument()
  })
})
