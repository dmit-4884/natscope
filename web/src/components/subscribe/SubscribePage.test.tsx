import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, within } from '@/test/utils'
import { clearAllSubscribeDrafts } from '@/stores/subscribeDraftStore'
import type { LiveMessage } from '../messages/unified/messageListUtils'
import { useLiveSubscription } from '../messages/unified/useLiveSubscription'
import SubscribePage from './SubscribePage'

vi.mock('react-router-dom', async (importOriginal) => ({
  ...(await importOriginal<typeof import('react-router-dom')>()),
  useOutletContext: () => ({ connectionId: 'conn-1', handleOpenMappings: vi.fn() }),
}))

vi.mock('@/contexts/live', () => ({
  useLiveStatsStore: (pick: (s: { stats: null }) => unknown) => pick({ stats: null }),
}))

const mappingItems = vi.hoisted(() => ({ data: [] as { pattern: string }[] }))

vi.mock('@/contexts/mappings', () => ({
  useMappingItems: () => mappingItems,
}))

vi.mock('@/contexts/settings', () => ({
  useLivePolicy: () => ({ maxDisplayRate: 0 }),
  useUpdateSettings: () => ({ mutate: vi.fn() }),
  useDisplayPreferences: () => ({ density: 'comfortable', timestampFormat: 'relative' }),
}))

vi.mock('../messages/unified/useLiveSubscription', () => ({
  useLiveSubscription: vi.fn(),
}))

vi.mock('../messages/UnifiedMessageViewer', () => ({
  default: () => <div data-testid="viewer" />,
}))

vi.mock('../messages/unified/MessageVirtualTable', () => ({
  MessageVirtualTable: ({ messages }: { messages: LiveMessage[] }) => (
    <ul data-testid="feed">
      {messages.map((m) => (
        <li key={m.id}>{m.subject}</li>
      ))}
    </ul>
  ),
}))

const mockedLive = vi.mocked(useLiveSubscription)

function liveState(over: Partial<ReturnType<typeof useLiveSubscription>> = {}): ReturnType<typeof useLiveSubscription> {
  return {
    liveMessages: [],
    liveLimit: 100,
    setLiveLimit: vi.fn(),
    wsStatus: 'connected',
    wsError: null,
    isPaused: false,
    togglePause: vi.fn(),
    newMessageIds: new Set(),
    clearMessages: vi.fn(),
    subjectCounts: {},
    deniedSubjects: [],
    ...over,
  }
}

function message(id: string, subject: string): LiveMessage {
  return { id, stream_name: '', subject, timestamp: Date.now(), data_base64: '', data_size: 0, content_type: 'text', headers: {} }
}

const input = () => screen.getByLabelText('Subjects')
const chips = () => screen.queryAllByTestId('subject-chip')

function addSubject(subject: string) {
  fireEvent.change(input(), { target: { value: subject } })
  fireEvent.keyDown(input(), { key: 'Enter' })
}

describe('SubscribePage', () => {
  beforeEach(() => {
    clearAllSubscribeDrafts()
    mappingItems.data = []
    mockedLive.mockReset()
    mockedLive.mockReturnValue(liveState())
  })

  it('starts on Cmd/Ctrl+Enter while suggestions are open', () => {
    mappingItems.data = [{ pattern: 'audit.>' }]
    render(<SubscribePage />)
    addSubject('orders.>')
    fireEvent.focus(input())
    fireEvent.keyDown(input(), { key: 'Enter', metaKey: true })

    expect(mockedLive).toHaveBeenLastCalledWith(expect.objectContaining({ subjects: ['orders.>'], enabled: true }))
    expect(input()).toHaveValue('')
  })

  it('does not pick a suggestion on Enter in the empty field', () => {
    mappingItems.data = [{ pattern: 'audit.>' }]
    render(<SubscribePage />)
    fireEvent.focus(input())
    fireEvent.keyDown(input(), { key: 'Enter' })

    expect(input()).toHaveValue('')
    fireEvent.keyDown(input(), { key: 'ArrowDown' })
    fireEvent.keyDown(input(), { key: 'Enter' })
    expect(input()).toHaveValue('audit.>')
  })

  it('explains core NATS before the first subscription', () => {
    render(<SubscribePage />)
    expect(screen.getByText('Subscribe to any subject')).toBeInTheDocument()
    expect(screen.getByTestId('subscribe-status')).toHaveTextContent('Stopped')
  })

  it('asks for a subject instead of starting with none', () => {
    render(<SubscribePage />)
    fireEvent.click(screen.getByRole('button', { name: 'Start' }))
    expect(screen.getByTestId('subject-error')).toHaveTextContent('Add a subject to subscribe to')
    expect(mockedLive).not.toHaveBeenCalledWith(expect.objectContaining({ enabled: true }))
  })

  it('adds subjects as chips on Enter and removes them with Backspace', () => {
    render(<SubscribePage />)
    addSubject('orders.>')
    addSubject('payments.*')
    expect(chips().map((c) => c.textContent)).toEqual(['orders.>', 'payments.*'])

    fireEvent.keyDown(input(), { key: 'Backspace' })
    expect(chips().map((c) => c.textContent)).toEqual(['orders.>'])
  })

  it('rejects an invalid subject inline', () => {
    render(<SubscribePage />)
    addSubject('orders..x')
    expect(screen.getByTestId('subject-error')).toBeInTheDocument()
    expect(input()).toHaveAttribute('aria-invalid', 'true')
    expect(chips()).toHaveLength(0)
  })

  it('starts the subscription with the typed subject and waits for messages', () => {
    render(<SubscribePage />)
    fireEvent.change(input(), { target: { value: 'orders.>' } })
    fireEvent.click(screen.getByRole('button', { name: 'Start' }))

    expect(mockedLive).toHaveBeenLastCalledWith(expect.objectContaining({ subjects: ['orders.>'], enabled: true }))
    expect(screen.getByTestId('subscribe-status')).toHaveTextContent('Live')
    expect(screen.getByText('Waiting for messages…')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Stop' })).toBeInTheDocument()
  })

  it('marks a refused subject while the others keep receiving', () => {
    mockedLive.mockReturnValue(liveState({ deniedSubjects: ['secret.>'], liveMessages: [message('1', 'orders.new')] }))
    render(<SubscribePage />)
    addSubject('orders.>')
    addSubject('secret.>')
    fireEvent.click(screen.getByRole('button', { name: 'Start' }))

    const denied = chips().find((c) => c.dataset.denied)
    expect(denied).toHaveTextContent('secret.>')
    expect(denied).toHaveTextContent('no permission')
    expect(chips().filter((c) => c.dataset.denied)).toHaveLength(1)
    expect(within(screen.getByTestId('feed')).getByText('orders.new')).toBeInTheDocument()
  })

  it('shows the missing permission when every subject is refused', () => {
    mockedLive.mockReturnValue(liveState({ deniedSubjects: ['secret.>'] }))
    render(<SubscribePage />)
    addSubject('secret.>')
    fireEvent.click(screen.getByRole('button', { name: 'Start' }))

    expect(screen.getByText('No permission to subscribe')).toBeInTheDocument()
    expect(screen.getByText(/subscribe to secret\.>/)).toBeInTheDocument()
    expect(screen.getByTestId('subscribe-status')).toHaveTextContent('No permission')
    expect(screen.queryByRole('button', { name: 'Pause' })).not.toBeInTheDocument()
  })

  it('keeps received messages after Stop and says how to resume', () => {
    mockedLive.mockReturnValue(liveState({ liveMessages: [message('1', 'orders.new')] }))
    render(<SubscribePage />)
    addSubject('orders.>')
    fireEvent.click(screen.getByRole('button', { name: 'Start' }))
    fireEvent.click(screen.getByRole('button', { name: 'Stop' }))

    expect(screen.getByTestId('subscribe-stopped')).toBeInTheDocument()
    expect(within(screen.getByTestId('feed')).getByText('orders.new')).toBeInTheDocument()
    expect(mockedLive).toHaveBeenLastCalledWith(expect.objectContaining({ enabled: false }))
  })

  it('offers quick-add presets and remembers started subjects as recents', () => {
    render(<SubscribePage />)
    fireEvent.change(input(), { target: { value: 'orders.>' } })
    fireEvent.click(screen.getByRole('button', { name: 'Start' }))
    fireEvent.click(screen.getByRole('button', { name: 'Stop' }))
    fireEvent.click(screen.getByRole('button', { name: 'Remove orders.>' }))

    const quickAdd = within(screen.getByText('Quick add:').parentElement!)
    expect(quickAdd.getByRole('button', { name: /all JetStream events/i })).toBeInTheDocument()
    fireEvent.click(quickAdd.getByRole('button', { name: 'orders.>' }))
    expect(chips().map((c) => c.textContent)).toEqual(['orders.>'])
  })
})
