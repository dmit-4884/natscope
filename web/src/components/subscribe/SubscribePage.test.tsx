import { useLayoutEffect, useState } from 'react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, within } from '@/test/utils'
import { clearAllSubscribeDrafts } from '@/stores/subscribeDraftStore'
import type { LiveMessage } from '../messages/unified/messageListUtils'
import { useLiveSubscription } from '../messages/unified/useLiveSubscription'
import { SubscribeSessionProvider } from './SubscribeSessionProvider'
import SubscribePage from './SubscribePage'

vi.mock('react-router-dom', async (importOriginal) => ({
  ...(await importOriginal<typeof import('react-router-dom')>()),
  useOutletContext: () => ({ connectionId: 'conn-1', handleOpenMappings: vi.fn() }),
}))

const mappingItems = vi.hoisted(() => ({ data: [] as { pattern: string }[] }))

vi.mock('@/contexts/mappings', () => ({
  useMappingItems: () => mappingItems,
}))

const updateSettings = vi.hoisted(() => vi.fn())
const livePolicy = vi.hoisted(() => ({ maxDisplayRate: 0 }))

vi.mock('@/contexts/settings', () => ({
  useLivePolicy: () => livePolicy,
  useUpdateSettings: () => ({ mutate: updateSettings }),
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
    msgPerSecond: undefined,
    messagesDropped: undefined,
    messagesReceived: undefined,
    pausedCount: 0,
    ...over,
  }
}

function message(id: string, subject: string): LiveMessage {
  return { id, stream_name: '', subject, timestamp: Date.now(), data_base64: '', data_size: 0, content_type: 'text', headers: {} }
}

function renderPage() {
  return render(
    <SubscribeSessionProvider connectionId="conn-1">
      <SubscribePage />
    </SubscribeSessionProvider>,
  )
}

function AwayAndBack() {
  const [here, setHere] = useState(true)
  return (
    <>
      <button type="button" onClick={() => setHere((v) => !v)}>
        Toggle page
      </button>
      {here ? <SubscribePage /> : <p>Another page</p>}
    </>
  )
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
    renderPage()
    addSubject('orders.>')
    fireEvent.focus(input())
    fireEvent.keyDown(input(), { key: 'Enter', metaKey: true })

    expect(mockedLive).toHaveBeenLastCalledWith(expect.objectContaining({ subjects: ['orders.>'], enabled: true }))
    expect(input()).toHaveValue('')
  })

  it('does not pick a suggestion on Enter in the empty field', () => {
    mappingItems.data = [{ pattern: 'audit.>' }]
    renderPage()
    fireEvent.focus(input())
    fireEvent.keyDown(input(), { key: 'Enter' })

    expect(input()).toHaveValue('')
    fireEvent.keyDown(input(), { key: 'ArrowDown' })
    fireEvent.keyDown(input(), { key: 'Enter' })
    expect(input()).toHaveValue('audit.>')
  })

  it('explains core NATS before the first subscription', () => {
    renderPage()
    expect(screen.getByText('Subscribe to any subject')).toBeInTheDocument()
    expect(screen.getByTestId('subscribe-status')).toHaveTextContent('Not subscribed')
  })

  it('asks for a subject instead of starting with none', () => {
    renderPage()
    fireEvent.click(screen.getByRole('button', { name: 'Start' }))
    expect(screen.getByTestId('subject-error')).toHaveTextContent('Add a subject to subscribe to')
    expect(mockedLive).not.toHaveBeenCalledWith(expect.objectContaining({ enabled: true }))
  })

  it('adds subjects as chips on Enter and removes them with Backspace', () => {
    renderPage()
    addSubject('orders.>')
    addSubject('payments.*')
    expect(chips().map((c) => c.textContent)).toEqual(['orders.>', 'payments.*'])

    fireEvent.keyDown(input(), { key: 'Backspace' })
    expect(chips().map((c) => c.textContent)).toEqual(['orders.>'])
  })

  it('rejects an invalid subject inline', () => {
    renderPage()
    addSubject('orders..x')
    expect(screen.getByTestId('subject-error')).toBeInTheDocument()
    expect(input()).toHaveAttribute('aria-invalid', 'true')
    expect(chips()).toHaveLength(0)
  })

  it('starts the subscription with the typed subject and waits for messages', () => {
    renderPage()
    fireEvent.change(input(), { target: { value: 'orders.>' } })
    fireEvent.click(screen.getByRole('button', { name: 'Start' }))

    expect(mockedLive).toHaveBeenLastCalledWith(expect.objectContaining({ subjects: ['orders.>'], enabled: true }))
    expect(screen.getByTestId('subscribe-status')).toHaveTextContent('Live')
    expect(screen.getByText('Waiting for messages…')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Stop' })).toBeInTheDocument()
  })

  it('keeps its header the same when the feed starts, so Stop lands where Start was', () => {
    renderPage()
    const header = () => screen.getByRole('heading', { name: 'Subscribe' }).closest('.border-b')
    const before = header()?.className
    fireEvent.change(input(), { target: { value: 'orders.>' } })
    fireEvent.click(screen.getByRole('button', { name: 'Start' }))

    expect(header()?.className).toBe(before)
    expect(screen.getByText(/Watch any subject over core NATS/)).toBeInTheDocument()
  })

  it('marks a refused subject while the others keep receiving', () => {
    mockedLive.mockReturnValue(liveState({ deniedSubjects: ['secret.>'], liveMessages: [message('1', 'orders.new')] }))
    renderPage()
    addSubject('orders.>')
    addSubject('secret.>')
    fireEvent.click(screen.getByRole('button', { name: 'Start' }))

    const denied = chips().find((c) => c.dataset.denied)
    expect(denied).toHaveTextContent('secret.>')
    expect(denied).toHaveTextContent('no permission')
    expect(chips().filter((c) => c.dataset.denied)).toHaveLength(1)
    expect(within(screen.getByTestId('feed')).getByText('orders.new')).toBeInTheDocument()
  })

  it('says the subscription is down once the client stopped trying', () => {
    mockedLive.mockReturnValue(liveState({ wsStatus: 'disconnected', wsError: 'Connection failed after max retries' }))
    renderPage()
    addSubject('orders.>')
    fireEvent.click(screen.getByRole('button', { name: 'Start' }))

    expect(screen.getByTestId('subscribe-status')).toHaveTextContent('Disconnected')
  })

  it('says the subscription is reconnecting while NATS is down', () => {
    mockedLive.mockReturnValue(liveState({ wsStatus: 'reconnecting' }))
    renderPage()
    addSubject('orders.>')
    fireEvent.click(screen.getByRole('button', { name: 'Start' }))

    expect(screen.getByTestId('subscribe-status')).toHaveTextContent('Reconnecting…')
  })

  it('shows the missing permission when every subject is refused', () => {
    mockedLive.mockReturnValue(liveState({ deniedSubjects: ['secret.>'] }))
    renderPage()
    addSubject('secret.>')
    fireEvent.click(screen.getByRole('button', { name: 'Start' }))

    expect(screen.getByText('No permission to subscribe')).toBeInTheDocument()
    expect(screen.getByText(/subscribe to secret\.>/)).toBeInTheDocument()
    expect(screen.getByTestId('subscribe-status')).toHaveTextContent('No permission')
    expect(screen.queryByRole('button', { name: 'Pause' })).not.toBeInTheDocument()
  })

  it('keeps received messages after Stop and says how to resume', () => {
    mockedLive.mockReturnValue(liveState({ liveMessages: [message('1', 'orders.new')] }))
    renderPage()
    addSubject('orders.>')
    fireEvent.click(screen.getByRole('button', { name: 'Start' }))
    fireEvent.click(screen.getByRole('button', { name: 'Stop' }))

    expect(screen.getByTestId('subscribe-stopped')).toBeInTheDocument()
    expect(within(screen.getByTestId('feed')).getByText('orders.new')).toBeInTheDocument()
    expect(mockedLive).toHaveBeenLastCalledWith(expect.objectContaining({ enabled: false }))
  })

  it('stops the subscription when another page opens and keeps its messages', () => {
    mockedLive.mockReturnValue(liveState({ liveMessages: [message('1', 'orders.new')] }))
    render(
      <SubscribeSessionProvider connectionId="conn-1">
        <AwayAndBack />
      </SubscribeSessionProvider>,
    )
    addSubject('orders.>')
    fireEvent.click(screen.getByRole('button', { name: 'Start' }))

    fireEvent.click(screen.getByRole('button', { name: 'Toggle page' }))
    expect(screen.getByText('Another page')).toBeInTheDocument()
    expect(mockedLive).toHaveBeenLastCalledWith(expect.objectContaining({ enabled: false }))

    fireEvent.click(screen.getByRole('button', { name: 'Toggle page' }))
    expect(screen.getByTestId('subscribe-stopped')).toBeInTheDocument()
    expect(chips().map((c) => c.textContent)).toEqual(['orders.>'])
    expect(within(screen.getByTestId('feed')).getByText('orders.new')).toBeInTheDocument()
  })

  it('drops the session in the first frame on another connection', () => {
    const frames: string[] = []
    function Frame() {
      useLayoutEffect(() => {
        frames.push(screen.queryByTestId('subscribe-status')?.textContent ?? '')
      })
      return null
    }
    const page = (connectionId: string) => (
      <SubscribeSessionProvider connectionId={connectionId}>
        <SubscribePage />
        <Frame />
      </SubscribeSessionProvider>
    )
    const view = render(page('conn-1'))
    addSubject('orders.>')
    fireEvent.click(screen.getByRole('button', { name: 'Start' }))
    frames.length = 0

    view.rerender(page('conn-2'))

    expect(frames[0]).not.toContain('Live')
  })

  it('keeps a stopped feed while another page is open', () => {
    mockedLive.mockReturnValue(liveState({ liveMessages: [message('1', 'orders.new')] }))
    render(
      <SubscribeSessionProvider connectionId="conn-1">
        <AwayAndBack />
      </SubscribeSessionProvider>,
    )
    addSubject('orders.>')
    fireEvent.click(screen.getByRole('button', { name: 'Start' }))
    fireEvent.click(screen.getByRole('button', { name: 'Stop' }))

    fireEvent.click(screen.getByRole('button', { name: 'Toggle page' }))
    fireEvent.click(screen.getByRole('button', { name: 'Toggle page' }))

    expect(screen.getByTestId('subscribe-stopped')).toBeInTheDocument()
    expect(within(screen.getByTestId('feed')).getByText('orders.new')).toBeInTheDocument()
  })

  it('mutes a noisy subject and lets it back in', () => {
    mockedLive.mockReturnValue(
      liveState({
        liveMessages: [message('1', 'metrics.cpu'), message('2', 'orders.new')],
        subjectCounts: { 'metrics.cpu': 1, 'orders.new': 1 },
      }),
    )
    renderPage()
    addSubject('>')
    fireEvent.click(screen.getByRole('button', { name: 'Start' }))

    fireEvent.click(screen.getByRole('button', { name: 'Mute metrics.cpu' }))
    expect(within(screen.getByTestId('feed')).queryByText('metrics.cpu')).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Mute metrics.cpu' })).not.toBeInTheDocument()
    expect(mockedLive).toHaveBeenLastCalledWith(
      expect.objectContaining({ exclude: ['metrics.cpu'], subjectLimits: expect.objectContaining({ exclude: ['metrics.cpu'] }) }),
    )

    fireEvent.click(screen.getByRole('button', { name: 'Unmute metrics.cpu' }))
    expect(within(screen.getByTestId('feed')).getByText('metrics.cpu')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Mute metrics.cpu' })).toBeInTheDocument()
  })

  it('opens the details only for a selected message', () => {
    mockedLive.mockReturnValue(liveState({ liveMessages: [message('1', 'orders.new')] }))
    renderPage()
    addSubject('orders.>')
    fireEvent.click(screen.getByRole('button', { name: 'Start' }))

    expect(screen.queryByTestId('viewer')).not.toBeInTheDocument()
  })

  it('keeps the display rate to this subscription', () => {
    Element.prototype.scrollIntoView = vi.fn()
    renderPage()
    addSubject('orders.>')
    fireEvent.click(screen.getByRole('button', { name: 'Start' }))
    fireEvent.click(screen.getByRole('button', { name: 'Display rate' }))
    fireEvent.click(screen.getByRole('option', { name: 'At most 5 msg/s' }))

    expect(mockedLive).toHaveBeenLastCalledWith(expect.objectContaining({ subjectLimits: expect.objectContaining({ maxDisplayRate: 5 }) }))
    expect(mockedLive.mock.lastCall?.[0].maxDisplayRate).toBeUndefined()
    expect(updateSettings).not.toHaveBeenCalled()
  })

  it('names a display rate from the settings that the list does not offer', () => {
    livePolicy.maxDisplayRate = 100
    try {
      Element.prototype.scrollIntoView = vi.fn()
      renderPage()
      addSubject('orders.>')
      fireEvent.click(screen.getByRole('button', { name: 'Start' }))

      expect(screen.getByRole('button', { name: 'Display rate' })).toHaveTextContent('At most 100 msg/s')
    } finally {
      livePolicy.maxDisplayRate = 0
    }
  })

  it('shortens a large paused count', () => {
    mockedLive.mockReturnValue(liveState({ isPaused: true, pausedCount: 1_234_567, liveMessages: [message('1', 'orders.new')] }))
    renderPage()
    addSubject('orders.>')
    fireEvent.click(screen.getByRole('button', { name: 'Start' }))

    expect(screen.getByRole('button', { name: 'Resume (+1.2M)' })).toBeInTheDocument()
  })

  it('shows how many messages wait while paused', () => {
    mockedLive.mockReturnValue(liveState({ isPaused: true, pausedCount: 8, liveMessages: [message('1', 'orders.new')] }))
    renderPage()
    addSubject('orders.>')
    fireEvent.click(screen.getByRole('button', { name: 'Start' }))

    expect(screen.getByRole('button', { name: 'Resume (+8)' })).toBeInTheDocument()
  })

  it('counts received messages as the server does, not by the subjects it lists', () => {
    mockedLive.mockReturnValue(
      liveState({
        liveMessages: [message('1', 'orders.new')],
        subjectCounts: { 'orders.new': 250 },
        messagesReceived: 800_000,
      }),
    )
    renderPage()
    addSubject('orders.>')
    fireEvent.click(screen.getByRole('button', { name: 'Start' }))

    expect(screen.getByTestId('feed-counts')).toHaveTextContent(/^800,000 messages received/)
  })

  it('says how much of the feed is shown and how much was skipped', () => {
    mockedLive.mockReturnValue(
      liveState({
        liveMessages: [message('1', 'orders.new')],
        subjectCounts: { 'orders.new': 250 },
        messagesDropped: 12,
      }),
    )
    renderPage()
    addSubject('orders.>')
    fireEvent.click(screen.getByRole('button', { name: 'Start' }))

    expect(screen.getByTestId('feed-counts')).toHaveTextContent('250 messages received · showing the last 100 · 12 skipped')
  })

  it('offers quick-add presets and remembers started subjects as recents', () => {
    renderPage()
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
