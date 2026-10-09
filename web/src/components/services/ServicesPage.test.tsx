import { describe, it, expect, vi, beforeEach } from 'vitest'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { act, render, screen, fireEvent, waitFor, within } from '@/test/utils'
import { listServices, type MicroDiscovery, type MicroEndpoint, type MicroService } from '@/api/discovery'
import { clearAllRequestDrafts, getRequestDraft } from '@/stores/requestDraftStore'
import { clearServiceSamples } from './serviceRates'
import ServicesPage from './ServicesPage'

vi.mock('react-router-dom', async (importOriginal) => ({
  ...(await importOriginal<typeof import('react-router-dom')>()),
  useOutletContext: () => ({ connectionId: 'conn-1' }),
}))

vi.mock('@/api/discovery', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/api/discovery')>()),
  listServices: vi.fn(),
}))

const policy = vi.hoisted(() => ({ readOnly: false }))

vi.mock('@/contexts/connection', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/contexts/connection')>()),
  useConnectionPolicy: () => ({ readOnly: policy.readOnly, label: null }),
}))

const mockedList = vi.mocked(listServices)

const allowed = { status: 'allowed', operation: 'publish', subject: '$SRV.INFO' } as const

function endpoint(over: Partial<MicroEndpoint> = {}): MicroEndpoint {
  return {
    name: 'add',
    subject: 'calc.add',
    queue_group: 'q',
    metadata: {},
    stats: { num_requests: 10, num_errors: 1, last_error: 'boom', processing_time_ns: 5000, average_processing_time_ns: 500 },
    ...over,
  }
}

function service(over: Partial<MicroService> = {}): MicroService {
  const endpoints = over.endpoints ?? [endpoint()]
  return {
    name: 'calc',
    description: 'Adds numbers',
    versions: ['1.0.0'],
    instances: [{ id: 'instance-0001', version: '1.0.0', metadata: {}, started: Date.now() - 60_000, endpoints, rtt_ns: 2_000_000 }],
    endpoints,
    ...over,
  }
}

function discovery(over: Partial<MicroDiscovery> = {}): MicroDiscovery {
  return { info_access: allowed, stats_access: { ...allowed, subject: '$SRV.STATS' }, services: [], ...over }
}

function renderAt(path = '/services') {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route path="/services/:name?" element={<ServicesPage />} />
        <Route path="/request" element={<p>Request page</p>} />
        <Route path="/settings/connections/:id/edit" element={<p>Connection settings</p>} />
      </Routes>
    </MemoryRouter>,
  )
}

const servicesNav = () => screen.getByRole('navigation', { name: 'Services' })

describe('ServicesPage', () => {
  beforeEach(() => {
    mockedList.mockReset()
    clearAllRequestDrafts()
    clearServiceSamples()
  })

  it('names the missing permission and stops auto-refresh when discovery is denied', async () => {
    mockedList.mockResolvedValue(discovery({ info_access: { status: 'denied', operation: 'publish', subject: '$SRV.INFO' } }))
    renderAt()

    expect(await screen.findByText('No access to services')).toBeInTheDocument()
    expect(screen.getByText(/publish to \$SRV\.INFO/)).toBeInTheDocument()
    expect(screen.queryByRole('switch')).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Refresh' })).not.toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: /check again/i }))
    await waitFor(() => expect(mockedList).toHaveBeenCalledTimes(2))
  })

  it('points a refused reply inbox to the inbox prefix setting', async () => {
    mockedList.mockResolvedValue(discovery({ info_access: { status: 'denied', operation: 'subscribe', subject: '_INBOX.>' } }))
    renderAt()

    fireEvent.click(await screen.findByRole('link', { name: 'connection settings' }))
    expect(screen.getByText('Connection settings')).toBeInTheDocument()
  })

  it('explains an empty result with a demo service and offers to look again', async () => {
    mockedList.mockResolvedValue(discovery())
    renderAt()

    expect(await screen.findByText('No services found')).toBeInTheDocument()
    expect(screen.getByText('nats micro serve demo')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: /look again/i }))
    await waitFor(() => expect(mockedList).toHaveBeenCalledTimes(2))
  })

  it('opens the first service and names the instance behind each last error', async () => {
    mockedList.mockResolvedValue(discovery({ services: [service()] }))
    renderAt()

    const detail = await screen.findByTestId('service-detail')
    expect(within(detail).getByRole('heading', { name: 'calc' })).toBeInTheDocument()
    expect(within(detail).getByText('calc.add')).toBeInTheDocument()
    expect(within(screen.getByRole('list', { name: 'Last errors' })).getByText(/on …e-0001 \(v1\.0\.0\)/)).toBeInTheDocument()
    expect(within(servicesNav()).getByRole('button', { name: /calc/ })).toHaveAttribute('aria-current', 'page')
    expect(screen.getByTestId('services-summary')).toHaveTextContent('1 service · 1 instance')
  })

  it('opens the service named in the address', async () => {
    mockedList.mockResolvedValue(discovery({ services: [service(), service({ name: 'users', endpoints: [endpoint({ name: 'get', subject: 'users.get' })] })] }))
    renderAt('/services/users')

    expect(within(await screen.findByTestId('service-detail')).getByRole('heading', { name: 'users' })).toBeInTheDocument()
  })

  it('hides the statistics columns when statistics are denied', async () => {
    mockedList.mockResolvedValue(
      discovery({
        stats_access: { status: 'denied', operation: 'publish', subject: '$SRV.STATS' },
        services: [service({ endpoints: [endpoint({ stats: undefined })] })],
      }),
    )
    renderAt()

    expect(await screen.findByTestId('service-detail')).toBeInTheDocument()
    expect(screen.getByTestId('stats-denied')).toHaveTextContent('publish to $SRV.STATS')
    expect(screen.queryByRole('columnheader', { name: 'Req/s' })).not.toBeInTheDocument()
    expect(screen.queryByTestId('service-stats')).not.toBeInTheDocument()
  })

  it('says when statistics could not be read', async () => {
    mockedList.mockResolvedValue(
      discovery({ stats_access: { status: 'unknown', operation: 'publish', subject: '$SRV.STATS' }, services: [service()] }),
    )
    renderAt()

    expect(await screen.findByTestId('stats-unavailable')).toBeInTheDocument()
  })

  it('skips the denied statistics on auto-refresh and asks again on Refresh', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true })
    try {
      const statsDenied = { status: 'denied', operation: 'publish', subject: '$SRV.STATS' } as const
      const services = [service({ endpoints: [endpoint({ stats: undefined })] })]
      mockedList.mockResolvedValue(discovery({ stats_access: statsDenied, services }))
      renderAt()

      await screen.findByTestId('stats-denied')
      expect(mockedList).toHaveBeenLastCalledWith('conn-1', { skipStats: false }, expect.anything())

      mockedList.mockResolvedValue(discovery({ stats_access: undefined, services }))
      await act(() => vi.advanceTimersByTimeAsync(5_000))
      await waitFor(() => expect(mockedList).toHaveBeenCalledTimes(2))
      expect(mockedList).toHaveBeenLastCalledWith('conn-1', { skipStats: true }, expect.anything())
      expect(screen.getByTestId('stats-denied')).toBeInTheDocument()

      fireEvent.click(screen.getByRole('button', { name: 'Refresh' }))
      await waitFor(() => expect(mockedList).toHaveBeenCalledTimes(3))
      expect(mockedList).toHaveBeenLastCalledWith('conn-1', { skipStats: false }, expect.anything())
    } finally {
      vi.useRealTimers()
    }
  })

  it('turns two refreshes into a request rate and flags a failing service', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true })
    try {
      const at = (requests: number, errors: number) =>
        discovery({ services: [service({ endpoints: [endpoint({ stats: { num_requests: requests, num_errors: errors, last_error: '', processing_time_ns: requests * 1000, average_processing_time_ns: 1000 } })] })] })
      mockedList.mockResolvedValue(at(10, 1))
      renderAt()
      await screen.findByTestId('service-detail')
      expect(screen.getByTestId('service-stats')).toHaveTextContent('Requests…')

      mockedList.mockResolvedValue(at(20, 6))
      await act(() => vi.advanceTimersByTimeAsync(5_000))

      await waitFor(() => expect(screen.getByTestId('service-stats')).toHaveTextContent(/Requests\d+(\.\d)? req\/s/))
      expect(screen.getByTestId('service-stats')).toHaveTextContent('Errors50%')
      expect(within(screen.getByTestId('service-detail')).getByTestId('service-health')).toHaveTextContent('Failing')
      expect(screen.getByTestId('services-summary')).toHaveTextContent('1 failing')
    } finally {
      vi.useRealTimers()
    }
  })

  it('lets the keyboard reach the totals behind the request and error rates', async () => {
    mockedList.mockResolvedValue(discovery({ services: [service()] }))
    renderAt()

    const stats = await screen.findByTestId('service-stats')
    const focusable = Array.from(stats.querySelectorAll<HTMLElement>('[tabindex="0"]')).map((el) => el.textContent)
    expect(focusable).toEqual([expect.stringMatching(/^Requests/), expect.stringMatching(/^Errors/)])
  })

  it('keeps the last answers when a refresh fails', async () => {
    mockedList.mockResolvedValueOnce(discovery({ services: [service()] }))
    mockedList.mockRejectedValueOnce(new Error('connection lost'))
    renderAt()
    await screen.findByTestId('service-detail')

    fireEvent.click(screen.getByRole('button', { name: 'Refresh' }))

    expect(await screen.findByTestId('services-refresh-failed')).toHaveTextContent('connection lost')
    expect(screen.getByTestId('service-detail')).toBeInTheDocument()
  })

  it('keeps Refresh focusable while it refreshes and ignores clicks meanwhile', async () => {
    mockedList.mockResolvedValueOnce(discovery({ services: [service()] }))
    mockedList.mockReturnValueOnce(new Promise(() => {}))
    renderAt()
    await screen.findByTestId('service-detail')
    const refresh = screen.getByRole('button', { name: 'Refresh' })

    fireEvent.click(refresh)
    fireEvent.click(refresh)

    expect(refresh).not.toBeDisabled()
    expect(refresh).toHaveAttribute('aria-disabled', 'true')
    expect(mockedList).toHaveBeenCalledTimes(2)
  })

  it('filters the list without changing the open service', async () => {
    mockedList.mockResolvedValue(
      discovery({ services: [service(), service({ name: 'users', description: 'Profiles', endpoints: [endpoint({ name: 'get', subject: 'users.get' })] })] }),
    )
    renderAt()
    await screen.findByTestId('service-detail')

    fireEvent.change(screen.getByPlaceholderText('Filter by name or subject'), { target: { value: 'users.get' } })
    await waitFor(() => expect(within(servicesNav()).queryByRole('button', { name: /calc/ })).not.toBeInTheDocument())
    expect(within(screen.getByTestId('service-detail')).getByRole('heading', { name: 'calc' })).toBeInTheDocument()

    fireEvent.click(within(servicesNav()).getByRole('button', { name: /users/ }))
    expect(within(screen.getByTestId('service-detail')).getByRole('heading', { name: 'users' })).toBeInTheDocument()
  })

  it('prefills Request / Reply with the endpoint subject and proto types on Call', async () => {
    const proto_method = { source_id: 'src', service: 'calc.v1.Calc', method: 'Add', input_type: 'calc.v1.AddRequest', output_type: 'calc.v1.AddReply' }
    mockedList.mockResolvedValue(discovery({ services: [service({ endpoints: [endpoint({ proto_method })] })] }))
    renderAt()

    fireEvent.click(await screen.findByRole('button', { name: 'Call add' }))

    expect(screen.getByText('Request page')).toBeInTheDocument()
    const draft = getRequestDraft('conn-1')
    expect(draft.subject).toBe('calc.add')
    expect(draft.requestTypes['calc.add']).toEqual({ messageType: 'calc.v1.AddRequest', sourceId: 'src' })
  })

  it('offers no Call on a read-only connection', async () => {
    policy.readOnly = true
    try {
      mockedList.mockResolvedValue(discovery({ services: [service()] }))
      renderAt()

      await screen.findByTestId('service-detail')
      expect(screen.queryByRole('button', { name: 'Call add' })).not.toBeInTheDocument()
    } finally {
      policy.readOnly = false
    }
  })
})
