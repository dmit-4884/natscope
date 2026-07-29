import { describe, it, expect, vi } from 'vitest'
import userEvent from '@testing-library/user-event'
import { render, screen } from '@/test/utils'
import type { Message } from '@/types/nats'
import ExportDialog from './ExportDialog'

const createMessage = (overrides: Partial<Message> = {}): Message => ({
  sequence: 1,
  subject: 'test.subject',
  timestamp: Date.now(),
  data_base64: btoa('{"key": "value"}'),
  data_size: 16,
  content_type: 'json',
  ...overrides,
})

describe('ExportDialog', () => {
  it('renders when open', () => {
    const msg = createMessage()
    render(
      <ExportDialog
        isOpen={true}
        onClose={vi.fn()}
        messages={[msg]}
        streamName="test-stream"
      />
    )
    expect(screen.getByText('Export Messages')).toBeDefined()
  })

  it('does not render when closed', () => {
    render(
      <ExportDialog
        isOpen={false}
        onClose={vi.fn()}
        messages={[]}
        streamName="test-stream"
      />
    )
    expect(screen.queryByText('Export Messages')).toBeNull()
  })

  it('shows format options', () => {
    render(
      <ExportDialog
        isOpen={true}
        onClose={vi.fn()}
        messages={[createMessage()]}
        streamName="test-stream"
      />
    )
    expect(screen.getByText('JSON')).toBeDefined()
    expect(screen.getByText('NDJSON')).toBeDefined()
    expect(screen.getByText('CSV')).toBeDefined()
  })

  it('shows include decoded checkbox', () => {
    const msg = createMessage({ decoded: { key: 'decoded_value' }, decoded_type: 'my.Type' })
    render(
      <ExportDialog
        isOpen={true}
        onClose={vi.fn()}
        messages={[msg]}
        streamName="test-stream"
      />
    )
    expect(screen.getByText(/decoded protobuf/i)).toBeDefined()
  })

  it('shows export count', () => {
    render(
      <ExportDialog
        isOpen={true}
        onClose={vi.fn()}
        messages={[createMessage(), createMessage({ sequence: 2 })]}
        streamName="test-stream"
      />
    )
    expect(screen.getByText('2')).toBeDefined()
  })

  it('calls onClose when Cancel is clicked', async () => {
    const user = userEvent.setup()
    const onClose = vi.fn()
    render(
      <ExportDialog
        isOpen={true}
        onClose={onClose}
        messages={[createMessage()]}
        streamName="test-stream"
      />
    )
    await user.click(screen.getByText('Cancel'))
    expect(onClose).toHaveBeenCalled()
  })
})
