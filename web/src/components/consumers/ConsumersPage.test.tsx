import { describe, it, expect, vi, beforeEach } from 'vitest'
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom'
import { render, screen, fireEvent, waitFor, within } from '@/test/utils'
import { getConsumersOverview, type ConsumersOverview } from '@/api/stats'
import { AccessDeniedError } from '@/shared/domain/access'
import type { ConsumerInfo, StreamInfo } from '@/types/nats'
import { downloadBlob } from '@/utils/download'
import ConsumersPage from './ConsumersPage'

vi.mock('react-router-dom', async (importOriginal) => ({
  ...(await importOriginal<typeof import('react-router-dom')>()),
  useOutletContext: () => ({ connectionId: 'conn-1' }),
}))

vi.mock('@/api/stats', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/api/stats')>()),
  getConsumersOverview: vi.fn(),
}))

vi.mock('@/utils/download', () => ({ downloadBlob: vi.fn() }))

const mockedOverview = vi.mocked(getConsumersOverview)

function consumer(name: string, stream: string, overrides: Partial<ConsumerInfo> = {}): ConsumerInfo {
  return {
    name,
    stream_name: stream,
    num_pending: 0,
    num_ack_pending: 0,
    config: { durable_name: name, ack_policy: 'explicit', max_ack_pending: 100 },
    delivered: { consumer_seq: 1, stream_seq: 1, last_active: Date.now() - 2_000 },
    ack_floor: { consumer_seq: 1, stream_seq: 1 },
    ...overrides,
  }
}

function stream(name: string): StreamInfo {
  return {
    name,
    subjects: [`${name.toLowerCase()}.>`],
    messages: 10,
    bytes: 100,
    consumer_count: 1,
    created: Date.now(),
    config: { retention: 'limits', max_msgs: -1, max_bytes: -1, max_age: 0, discard: 'old' },
  }
}

function overview(over: Partial<ConsumersOverview> = {}): ConsumersOverview {
  return {
    consumers: [
      consumer('archiver', 'ORDERS'),
      consumer('billing', 'ORDERS', { num_pending: 40, num_ack_pending: 100, config: { durable_name: 'billing', max_ack_pending: 100, filter_subject: 'orders.created' } }),
      consumer('mailer', 'EVENTS', { num_pending: 7, num_ack_pending: 2, num_waiting: 2, num_redelivered: 2 }),
    ],
    streams: [stream('ORDERS'), stream('EVENTS')],
    unreadable: [],
    ...over,
  }
}

function Location() {
  const location = useLocation()
  return <p data-testid="location">{location.pathname + location.search}</p>
}

function renderPage() {
  return render(
    <MemoryRouter initialEntries={['/consumers']}>
      <Routes>
        <Route path="/consumers" element={<ConsumersPage />} />
        <Route path="/streams/:streamName/consumers" element={<Location />} />
      </Routes>
    </MemoryRouter>,
  )
}

const rows = () => screen.getAllByRole('button', { name: /ORDERS|EVENTS/ }).filter((el) => el.tagName === 'TR')

describe('ConsumersPage', () => {
  beforeEach(() => {
    mockedOverview.mockReset()
    vi.mocked(downloadBlob).mockReset()
  })

  it('lists every consumer, stuck ones first, with a summary', async () => {
    mockedOverview.mockResolvedValue(overview())
    renderPage()

    await screen.findByText('billing')
    expect(screen.getByTestId('consumers-summary')).toHaveTextContent('3 consumers on 2 streams · 1 stuck')
    expect(rows().map((r) => within(r).getAllByText(/^(archiver|billing|mailer)$/)[0].textContent)).toEqual(['billing', 'mailer', 'archiver'])
    expect(within(rows()[0]).getByText('Ack limit reached')).toBeInTheDocument()
    expect(within(rows()[2]).getByText('Caught up')).toBeInTheDocument()
  })

  it('opens a consumer on its stream page', async () => {
    mockedOverview.mockResolvedValue(overview())
    renderPage()

    fireEvent.click(await screen.findByText('billing'))
    expect(screen.getByTestId('location')).toHaveTextContent('/streams/ORDERS/consumers?consumer=billing')
  })

  it('filters by text and by problems', async () => {
    mockedOverview.mockResolvedValue(overview())
    renderPage()
    await screen.findByText('billing')

    fireEvent.click(screen.getByRole('switch', { name: /problems only/i }))
    await waitFor(() => expect(screen.queryByText('archiver')).not.toBeInTheDocument())
    expect(screen.getByText('mailer')).toBeInTheDocument()

    fireEvent.change(screen.getByPlaceholderText(/filter by consumer/i), { target: { value: 'orders.created' } })
    await waitFor(() => expect(screen.queryByText('mailer')).not.toBeInTheDocument())
    expect(screen.getByText('billing')).toBeInTheDocument()
  })

  it('names the streams whose consumers this user cannot see', async () => {
    mockedOverview.mockResolvedValue(
      overview({
        unreadable: [
          { stream: 'SECRET', access: { status: 'denied', operation: 'publish', subject: '$JS.API.CONSUMER.LIST.SECRET' } },
          { stream: 'BROKEN', error: 'stream offline' },
        ],
      }),
    )
    renderPage()

    const notice = await screen.findByTestId('consumers-unreadable')
    expect(notice).toHaveTextContent('Consumers of 2 streams are not shown')
    expect(notice).toHaveTextContent('SECRET')
    expect(notice).toHaveTextContent('publish to $JS.API.CONSUMER.LIST.SECRET')
    expect(notice).toHaveTextContent('stream offline')
  })

  it('shows the missing permission when streams cannot be listed at all', async () => {
    mockedOverview.mockRejectedValue(
      new AccessDeniedError('refused', { status: 'denied', operation: 'publish', subject: '$JS.API.STREAM.LIST' }),
    )
    renderPage()

    expect(await screen.findByText('No access to consumers')).toBeInTheDocument()
    expect(screen.getByText(/publish to \$JS\.API\.STREAM\.LIST/)).toBeInTheDocument()
    expect(screen.queryByRole('switch', { name: /auto-refresh/i })).not.toBeInTheDocument()
  })

  it('explains an empty account', async () => {
    mockedOverview.mockResolvedValue(overview({ consumers: [] }))
    renderPage()

    expect(await screen.findByText('No consumers yet')).toBeInTheDocument()
  })

  it('exports the shown consumers as CSV or JSON', async () => {
    mockedOverview.mockResolvedValue(overview())
    renderPage()
    await screen.findByText('billing')

    fireEvent.click(screen.getByRole('button', { name: 'Export consumers' }))
    fireEvent.click(await screen.findByRole('menuitem', { name: /csv/i }))
    expect(downloadBlob).toHaveBeenCalledWith(expect.stringContaining('ORDERS,billing,pull'), expect.stringMatching(/^consumers-.*\.csv$/), 'text/csv')

    fireEvent.click(screen.getByRole('button', { name: 'Export consumers' }))
    fireEvent.click(await screen.findByRole('menuitem', { name: /json/i }))
    expect(downloadBlob).toHaveBeenLastCalledWith(expect.stringContaining('"consumer": "billing"'), expect.stringMatching(/^consumers-.*\.json$/), 'application/json')
  })
})

describe('ConsumersPage rows', () => {
  beforeEach(() => {
    mockedOverview.mockReset()
  })

  it('names each row by consumer, stream and state', async () => {
    mockedOverview.mockResolvedValue(overview())
    renderPage()

    expect(await screen.findByRole('button', { name: 'billing on ORDERS: Ack limit reached' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'mailer on EVENTS: Redelivering' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'archiver on ORDERS: Caught up' })).toBeInTheDocument()
  })

  it('tells a forgotten delivery time from a consumer that never delivered', async () => {
    mockedOverview.mockResolvedValue(
      overview({
        consumers: [
          consumer('restarted', 'ORDERS', { delivered: { consumer_seq: 4, stream_seq: 4 } }),
          consumer('fresh', 'ORDERS', { delivered: { consumer_seq: 0, stream_seq: 9 } }),
        ],
      }),
    )
    renderPage()

    expect(within(await screen.findByRole('button', { name: /^restarted on/ })).getByText('unknown')).toBeInTheDocument()
    expect(within(screen.getByRole('button', { name: /^fresh on/ })).getByText('never')).toBeInTheDocument()
  })
})
