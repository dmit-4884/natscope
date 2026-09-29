import { describe, it, expect, vi, beforeEach } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import { render, screen, fireEvent, within } from '@/test/utils'
import { getStreamRelations } from '@/api/streams'
import { safeGetItem, safeRemoveItem, safeSetItem } from '@/utils/safeStorage'
import type { StreamInfo, StreamRelations } from '@/types/nats'
import StreamRelationsTab from './StreamRelationsTab'

vi.mock('react-router-dom', async (importOriginal) => ({
  ...(await importOriginal<typeof import('react-router-dom')>()),
  useOutletContext: () => ({ connectionId: 'conn-1', streamName: 'AGG' }),
}))

vi.mock('@/api/streams', () => ({
  getStreamRelations: vi.fn(),
}))

vi.mock('@/utils/safeStorage', () => ({
  safeGetItem: vi.fn(),
  safeSetItem: vi.fn(),
  safeRemoveItem: vi.fn(),
}))

const mockedGet = vi.mocked(getStreamRelations)

function info(name: string, messages: number): StreamInfo {
  return {
    name,
    subjects: [`${name.toLowerCase()}.>`],
    messages,
    bytes: 2048,
    consumer_count: 2,
    created: 0,
    config: { retention: 'limits', max_msgs: -1, max_bytes: -1, max_age: 0, storage: 'file', num_replicas: 3 },
  }
}

const relations: StreamRelations = {
  nodes: [
    { id: 'AGG', name: 'AGG', kind: 'stream', info: info('AGG', 1200) },
    { id: 'BACKUP', name: 'BACKUP', kind: 'stream', info: info('BACKUP', 1200) },
    { id: 'EU', name: 'EU', kind: 'stream', info: info('EU', 700) },
    { id: 'GHOST', name: 'GHOST', kind: 'missing' },
    { id: 'external $JS.hub.API US', name: 'US', kind: 'external', external: { api_prefix: '$JS.hub.API', deliver_prefix: '' } },
  ],
  edges: [
    {
      kind: 'source',
      from: 'EU',
      to: 'AGG',
      source: { name: 'EU', subject_transforms: [{ src: 'eu.>', dest: 'agg.eu.>' }] },
      state: { lag: 12, active_ns: 2_000_000_000 },
    },
    { kind: 'source', from: 'GHOST', to: 'AGG', source: { name: 'GHOST' }, state: { lag: 0, active_ns: -1, error: 'stream not found' } },
    { kind: 'source', from: 'external $JS.hub.API US', to: 'AGG', source: { name: 'US' }, state: { lag: 0, active_ns: -1 } },
    { kind: 'mirror', from: 'AGG', to: 'BACKUP', source: { name: 'AGG' }, state: { lag: 0, active_ns: 500_000_000 } },
  ],
}

function renderTab() {
  return render(
    <MemoryRouter>
      <StreamRelationsTab />
    </MemoryRouter>,
  )
}

describe('StreamRelationsTab', () => {
  beforeEach(() => {
    mockedGet.mockReset()
    vi.mocked(safeGetItem).mockReturnValue(null)
    vi.mocked(safeSetItem).mockReset()
    vi.mocked(safeRemoveItem).mockReset()
  })

  it('draws the stream with its upstreams, downstreams and placeholders', async () => {
    mockedGet.mockResolvedValue(relations)
    renderTab()

    expect(await screen.findByText('5 nodes · 4 links')).toBeInTheDocument()
    expect(screen.getAllByTestId('relation-node')).toHaveLength(5)
    expect(screen.getByRole('link', { name: 'EU: show its relations' })).toHaveAttribute('href', '/streams/EU/relations')
    expect(screen.getByText('Not found on this connection')).toBeInTheDocument()
    expect(screen.getByText('Domain hub')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Source from EU to AGG. Show details' })).toHaveTextContent('Source· lag 12')
    expect(screen.getByRole('button', { name: 'Source from US to AGG. Show details' })).toHaveTextContent('never seen')
  })

  it('shows link details', async () => {
    mockedGet.mockResolvedValue(relations)
    renderTab()

    fireEvent.click(await screen.findByRole('button', { name: 'Source from GHOST to AGG. Show details' }))
    const details = screen.getByTestId('relation-details')
    expect(within(details).getByText('stream not found')).toBeInTheDocument()
    expect(within(details).getByText('never')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Source from EU to AGG. Show details' }))
    expect(within(screen.getByTestId('relation-details')).getByText('agg.eu.>')).toBeInTheDocument()
    expect(within(screen.getByTestId('relation-details')).getByText('12 messages behind')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Close link details' }))
    expect(screen.queryByTestId('relation-details')).toBeNull()
  })

  it('closes link details with Escape from the link label', async () => {
    mockedGet.mockResolvedValue(relations)
    renderTab()

    const label = await screen.findByRole('button', { name: 'Mirror from AGG to BACKUP. Show details' })
    fireEvent.click(label)
    expect(screen.getByTestId('relation-details')).toBeInTheDocument()

    fireEvent.keyDown(label, { key: 'Escape' })
    expect(screen.queryByTestId('relation-details')).toBeNull()
    expect(label).toHaveAttribute('aria-pressed', 'false')
  })

  it('keeps a moved details panel in place across links until docked back', async () => {
    vi.mocked(safeGetItem).mockImplementation((key) => (key === 'nats_relations_details_position' ? '{"x":40,"y":60}' : null))
    mockedGet.mockResolvedValue(relations)
    renderTab()

    fireEvent.click(await screen.findByRole('button', { name: 'Source from GHOST to AGG. Show details' }))
    expect(screen.getByTestId('relation-details')).toHaveStyle({ left: '40px', top: '60px' })

    fireEvent.click(screen.getByRole('button', { name: 'Mirror from AGG to BACKUP. Show details' }))
    expect(screen.getByTestId('relation-details')).toHaveStyle({ left: '40px', top: '60px' })

    fireEvent.doubleClick(screen.getByTestId('relation-details-handle'))
    expect(screen.getByTestId('relation-details').style.left).toBe('')
    expect(safeRemoveItem).toHaveBeenCalledWith('nats_relations_details_position')
  })

  it('limits the graph by depth and remembers it', async () => {
    mockedGet.mockResolvedValue({
      nodes: ['A', 'B', 'C', 'AGG'].map((id) => ({ id, name: id, kind: 'stream' as const, info: info(id, 1) })),
      edges: [
        { kind: 'source', from: 'A', to: 'B' },
        { kind: 'source', from: 'B', to: 'C' },
        { kind: 'source', from: 'C', to: 'AGG' },
      ],
    })
    renderTab()

    expect(await screen.findByText('4 nodes · 3 links')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Decrease depth' }))
    fireEvent.click(screen.getByRole('button', { name: 'Decrease depth' }))
    expect(screen.getByText('2 nodes · 1 link')).toBeInTheDocument()
    expect(safeSetItem).toHaveBeenLastCalledWith('nats_relations_depth', '1')
  })

  it('hides a link type from the legend', async () => {
    mockedGet.mockResolvedValue(relations)
    renderTab()

    fireEvent.click(await screen.findByRole('button', { name: 'Mirror' }))
    expect(screen.getByText('4 nodes · 3 links')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Mirror' })).toHaveAttribute('aria-pressed', 'false')
  })

  it('explains when the stream has no relations', async () => {
    mockedGet.mockResolvedValue({ nodes: [], edges: [] })
    renderTab()

    expect(await screen.findByText('No relations')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Open config' })).toBeInTheDocument()
  })
})
