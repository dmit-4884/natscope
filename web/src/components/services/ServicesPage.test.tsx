import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor, within } from '@/test/utils'
import { listServices, type MicroDiscovery, type MicroEndpoint, type MicroService } from '@/api/discovery'
import { clearAllRequestDrafts, getRequestDraft } from '@/stores/requestDraftStore'
import ServicesPage from './ServicesPage'

const navigate = vi.fn()

vi.mock('react-router-dom', async (importOriginal) => ({
  ...(await importOriginal<typeof import('react-router-dom')>()),
  useOutletContext: () => ({ connectionId: 'conn-1' }),
  useNavigate: () => navigate,
}))

vi.mock('@/api/discovery', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/api/discovery')>()),
  listServices: vi.fn(),
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
    instances: [{ id: 'inst-1', version: '1.0.0', metadata: {}, started: Date.now() - 60_000, endpoints }],
    endpoints,
    ...over,
  }
}

function discovery(over: Partial<MicroDiscovery> = {}): MicroDiscovery {
  return { info_access: allowed, stats_access: { ...allowed, subject: '$SRV.STATS' }, services: [], ...over }
}

describe('ServicesPage', () => {
  beforeEach(() => {
    mockedList.mockReset()
    navigate.mockReset()
    clearAllRequestDrafts()
  })

  it('names the missing permission and stops auto-refresh when discovery is denied', async () => {
    mockedList.mockResolvedValue(discovery({ info_access: { status: 'denied', operation: 'publish', subject: '$SRV.INFO' } }))
    render(<ServicesPage />)

    expect(await screen.findByText('No access to services')).toBeInTheDocument()
    expect(screen.getByText(/publish to \$SRV\.INFO/)).toBeInTheDocument()
    expect(screen.queryByRole('switch', { name: 'Auto-refresh' })).not.toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: /check again/i }))
    await waitFor(() => expect(mockedList).toHaveBeenCalledTimes(2))
  })

  it('explains an empty result and offers to look again', async () => {
    mockedList.mockResolvedValue(discovery())
    render(<ServicesPage />)

    expect(await screen.findByText('No services found')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: /look again/i }))
    await waitFor(() => expect(mockedList).toHaveBeenCalledTimes(2))
  })

  it('lists services with totals and opens the first one', async () => {
    mockedList.mockResolvedValue(discovery({ services: [service()] }))
    render(<ServicesPage />)

    const detail = await screen.findByTestId('service-detail')
    expect(within(detail).getByRole('heading', { name: 'calc' })).toBeInTheDocument()
    expect(within(detail).getByText('calc.add')).toBeInTheDocument()
    expect(within(detail).getByText('boom')).toBeInTheDocument()
    expect(screen.queryByTestId('stats-denied')).not.toBeInTheDocument()
  })

  it('keeps listing services when statistics are denied', async () => {
    mockedList.mockResolvedValue(
      discovery({
        stats_access: { status: 'denied', operation: 'publish', subject: '$SRV.STATS' },
        services: [service({ endpoints: [endpoint({ stats: undefined })] })],
      }),
    )
    render(<ServicesPage />)

    const banner = await screen.findByTestId('stats-denied')
    expect(banner).toHaveTextContent('publish to $SRV.STATS')
    expect(screen.getAllByLabelText('No access to statistics').length).toBeGreaterThan(0)
    expect(screen.getByTestId('service-detail')).toBeInTheDocument()
  })

  it('filters services by name, description or subject', async () => {
    mockedList.mockResolvedValue(
      discovery({ services: [service(), service({ name: 'users', description: 'Profiles', endpoints: [endpoint({ name: 'get', subject: 'users.get' })] })] }),
    )
    render(<ServicesPage />)
    await screen.findByTestId('service-detail')

    fireEvent.change(screen.getByPlaceholderText('Filter services'), { target: { value: 'users.get' } })
    await waitFor(() => expect(within(screen.getByTestId('service-detail')).getByRole('heading', { name: 'users' })).toBeInTheDocument())
    expect(screen.queryByRole('heading', { name: 'calc' })).not.toBeInTheDocument()
  })

  it('prefills Request / Reply with the endpoint subject and proto types on Call', async () => {
    const proto_method = { source_id: 'src', service: 'calc.v1.Calc', method: 'Add', input_type: 'calc.v1.AddRequest', output_type: 'calc.v1.AddReply' }
    mockedList.mockResolvedValue(discovery({ services: [service({ endpoints: [endpoint({ proto_method })] })] }))
    render(<ServicesPage />)

    fireEvent.click(await screen.findByRole('button', { name: 'Call add' }))

    expect(navigate).toHaveBeenCalledWith('/request')
    const draft = getRequestDraft('conn-1')
    expect(draft.subject).toBe('calc.add')
    expect(draft.requestTypes['calc.add']).toEqual({ messageType: 'calc.v1.AddRequest', sourceId: 'src' })
  })
})
