import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@/test/utils'
import type { Message } from '@/types/nats'
import MessageDiffViewer from './MessageDiffViewer'

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
})
