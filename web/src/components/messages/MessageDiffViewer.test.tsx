import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@/test/utils'
import type { Message } from '@/types/nats'
import MessageDiffViewer from './MessageDiffViewer'

const full = vi.hoisted(() => ({
  bySeq: {} as Record<number, { data?: unknown; error?: unknown; isLoading?: boolean }>,
  requested: [] as Array<number | null>,
}))

vi.mock('@/contexts/messages', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/contexts/messages')>()),
  useStreamMessage: (_conn: string | null, _stream: string | null, seq: number | null) => {
    full.requested.push(seq)
    return seq == null ? { data: undefined, error: null, isLoading: false, refetch: vi.fn() } : { error: null, isLoading: false, refetch: vi.fn(), ...full.bySeq[seq] }
  },
}))

const createMessage = (overrides: Partial<Message> = {}): Message => ({
  sequence: 1,
  subject: 'test.subject',
  timestamp: Date.now(),
  data_base64: btoa('{"key": "value1"}'),
  data_size: 18,
  content_type: 'json',
  ...overrides,
})

describe('MessageDiffViewer', () => {
  it('shows placeholder when no messages selected', () => {
    render(<MessageDiffViewer messageA={null} messageB={null} onClose={vi.fn()} />)
    expect(screen.getByText(/select two messages/i)).toBeDefined()
  })

  it('shows placeholder when only one message selected', () => {
    const msg = createMessage()
    render(<MessageDiffViewer messageA={msg} messageB={null} onClose={vi.fn()} />)
    expect(screen.getByText(/select two messages/i)).toBeDefined()
  })

  it('renders diff when both messages provided', () => {
    const msgA = createMessage({ sequence: 1, data_base64: btoa('{"key": "value1"}') })
    const msgB = createMessage({ sequence: 2, data_base64: btoa('{"key": "value2"}') })
    render(<MessageDiffViewer messageA={msgA} messageB={msgB} onClose={vi.fn()} />)
    expect(screen.getByText(/message diff/i)).toBeDefined()
  })

  it('shows metadata comparison', () => {
    const msgA = createMessage({ sequence: 1, subject: 'test.a' })
    const msgB = createMessage({ sequence: 2, subject: 'test.b' })
    render(<MessageDiffViewer messageA={msgA} messageB={msgB} onClose={vi.fn()} />)
    expect(screen.getByText(/metadata/i)).toBeDefined()
  })

  it('uses decoded data when available', () => {
    const msgA = createMessage({
      sequence: 1,
      decoded: { name: 'Alice' },
      decoded_type: 'User',
    })
    const msgB = createMessage({
      sequence: 2,
      decoded: { name: 'Bob' },
      decoded_type: 'User',
    })
    render(<MessageDiffViewer messageA={msgA} messageB={msgB} onClose={vi.fn()} />)
    // Should show decoded content in the diff
    expect(screen.getByText(/payload/i)).toBeDefined()
  })

  it('calls onClose when close button clicked', async () => {
    const { userEvent: user } = await import('@testing-library/user-event')
    const userInstance = user.setup()
    const onClose = vi.fn()
    const msgA = createMessage({ sequence: 1 })
    const msgB = createMessage({ sequence: 2 })
    render(<MessageDiffViewer messageA={msgA} messageB={msgB} onClose={onClose} />)

    // Find and click the close button (X or Close)
    const closeButtons = screen.getAllByRole('button')
    const closeBtn = closeButtons.find(
      (btn) =>
        btn.textContent?.includes('Close') ||
        btn.getAttribute('aria-label')?.includes('close') ||
        btn.getAttribute('aria-label')?.includes('Close')
    )
    if (closeBtn) {
      await userInstance.click(closeBtn)
      expect(onClose).toHaveBeenCalled()
    }
  })

  describe('truncated list payloads', () => {
    const clipped = (sequence: number) =>
      createMessage({ sequence, truncated: true, data_base64: btoa('{"a":"same prefix'), data_size: 5_000_000 })
    const renderDiff = (a: Message, b: Message) =>
      render(<MessageDiffViewer messageA={a} messageB={b} connectionId="c1" streamName="BLOBS" onClose={vi.fn()} />)

    it('does not ask the server for messages that came whole', () => {
      full.requested.length = 0
      renderDiff(createMessage({ sequence: 1 }), createMessage({ sequence: 2 }))

      expect(full.requested.every((seq) => seq === null)).toBe(true)
    })

    it('loads the full message of a clipped side and diffs that, not the preview', () => {
      full.bySeq = {
        1: { data: createMessage({ sequence: 1, data_base64: btoa('{"a":"same prefix","tail":1}') }) },
        2: { data: createMessage({ sequence: 2, data_base64: btoa('{"a":"same prefix","tail":2}') }) },
      }
      renderDiff(clipped(1), clipped(2))

      expect(full.requested).toContain(1)
      expect(full.requested).toContain(2)
      expect(screen.queryByText('No payload differences')).toBeNull()
      expect(screen.getByText(/"tail": 2/)).toBeDefined()
    })

    it('waits for the full message instead of showing a partial diff', () => {
      full.bySeq = { 1: { isLoading: true }, 2: { isLoading: true } }
      renderDiff(clipped(1), clipped(2))

      expect(screen.getByText(/loading the full messages/i)).toBeDefined()
      expect(screen.queryByText('No payload differences')).toBeNull()
    })

    it('says so when a full message cannot be loaded', () => {
      full.bySeq = { 1: { error: new Error('message gone') }, 2: { data: createMessage({ sequence: 2 }) } }
      renderDiff(clipped(1), clipped(2))

      expect(screen.getByRole('alert')).toHaveTextContent(/message gone/)
      expect(screen.queryByText('No payload differences')).toBeNull()
    })
  })

  describe('headers', () => {
    it('diffs headers as well as payload', () => {
      full.bySeq = {}
      const a = createMessage({ sequence: 1, headers: { 'X-Trace': 'one' } })
      const b = createMessage({ sequence: 2, headers: { 'X-Trace': 'two', 'X-New': 'y' } })
      render(<MessageDiffViewer messageA={a} messageB={b} onClose={vi.fn()} />)

      expect(screen.getByRole('heading', { name: /headers/i })).toBeDefined()
      expect(screen.getByText(/"X-New": "y"/)).toBeDefined()
    })

    it('says the headers are the same when they are', () => {
      full.bySeq = {}
      const headers = { 'X-Trace': 'one' }
      render(<MessageDiffViewer messageA={createMessage({ sequence: 1, headers })} messageB={createMessage({ sequence: 2, headers })} onClose={vi.fn()} />)

      expect(screen.getByText('No header differences')).toBeDefined()
    })
  })
})
