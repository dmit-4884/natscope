import { beforeEach, describe, it, expect, vi } from 'vitest'
import userEvent from '@testing-library/user-event'
import { fireEvent, render, screen, waitFor } from '@/test/utils'
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

describe('ExportDialog range export', () => {
  const download = vi.hoisted(() => ({ blobs: [] as Blob[], names: [] as string[] }))
  const api = vi.hoisted(() => ({ getMessages: vi.fn() }))
  const toasts = vi.hoisted(() => ({ success: vi.fn(), info: vi.fn(), warning: vi.fn(), error: vi.fn() }))

  vi.mock('@/api/messages', () => ({ getMessages: api.getMessages }))
  vi.mock('@/utils/toast', () => ({ toast: toasts }))
  vi.mock('@/contexts/settings', () => ({
    useMessagesPolicy: () => ({ defaultExportFormat: 'ndjson', exportRangeLimit: 50_000 }),
  }))

  const page = (from: number, count: number, hasMore: boolean) => ({
    messages: Array.from({ length: count }, (_, i) => createMessage({ sequence: from + i })),
    has_more: hasMore,
    next_seq: hasMore ? from + count : 0,
  })

  beforeEach(() => {
    download.blobs.length = 0
    api.getMessages.mockReset()
    Object.values(toasts).forEach((fn) => fn.mockReset())
    URL.createObjectURL = vi.fn((blob: Blob) => {
      download.blobs.push(blob)
      return 'blob:x'
    })
    URL.revokeObjectURL = vi.fn()
  })

  const renderRange = (props: Partial<React.ComponentProps<typeof ExportDialog>> = {}) =>
    render(<ExportDialog isOpen onClose={vi.fn()} messages={[]} streamName="ORDERS" connectionId="c1" streamFirstSeq={1} totalCount={5000} {...props} />)

  it('walks only the subject the toolbar filters on, and says so', async () => {
    api.getMessages.mockResolvedValue(page(1, 3, false))
    const user = userEvent.setup()
    renderRange({ subjectFilter: 'orders.eu.>' })

    await user.click(screen.getByTestId('export-scope-range'))
    expect(screen.getByText(/orders\.eu\.>/)).toBeDefined()
    await user.click(screen.getByTestId('export-confirm'))

    await waitFor(() => expect(api.getMessages).toHaveBeenCalled())
    expect(api.getMessages.mock.calls[0][1]).toMatchObject({ subject_filter: 'orders.eu.>', max_payload_bytes: 0 })
  })

  it('exports exactly the typed limit however the options were touched before', async () => {
    api.getMessages.mockImplementation(async (_s: string, params: { start_seq: number }) => page(params.start_seq, 500, true))
    const user = userEvent.setup()
    renderRange()

    await user.click(screen.getByTestId('export-scope-range'))
    fireEvent.change(screen.getByTestId('export-limit'), { target: { value: '620' } })
    await user.click(screen.getByText('CSV'))
    await user.click(screen.getByText('NDJSON'))
    await user.click(screen.getByTestId('export-confirm'))

    await waitFor(() => expect(download.blobs).toHaveLength(1))
    const text = await download.blobs[0].text()
    expect(text.trim().split('\n')).toHaveLength(620)
    expect(toasts.warning).toHaveBeenCalledWith(expect.stringContaining('620'))
  })

  it('saves what was fetched when Cancel lands inside a page fetch', async () => {
    let calls = 0
    api.getMessages.mockImplementation((_s: string, params: { start_seq: number }, signal?: AbortSignal) => {
      calls++
      if (calls === 1) return Promise.resolve(page(params.start_seq, 500, true))
      return new Promise((_resolve, reject) => {
        signal?.addEventListener('abort', () => reject(new DOMException('aborted', 'AbortError')))
      })
    })
    const user = userEvent.setup()
    renderRange()

    await user.click(screen.getByTestId('export-scope-range'))
    await user.click(screen.getByTestId('export-confirm'))
    await waitFor(() => expect(calls).toBe(2))
    await user.click(screen.getByTestId('export-cancel'))

    await waitFor(() => expect(download.blobs).toHaveLength(1))
    expect((await download.blobs[0].text()).trim().split('\n')).toHaveLength(500)
    expect(toasts.warning).toHaveBeenCalledWith(expect.stringMatching(/Export cancelled — 500 of 50,000 messages saved/))
  })

  it('caps the limit at one million and says the file is built in the browser', () => {
    renderRange()
    fireEvent.click(screen.getByTestId('export-scope-range'))

    fireEvent.change(screen.getByTestId('export-limit'), { target: { value: '5000000' } })

    expect(screen.getByTestId('export-limit')).toHaveValue(1_000_000)
    expect(screen.getByText(/built in the browser/i)).toBeDefined()
  })
})
