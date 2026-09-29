import { useState } from 'react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, act, waitFor } from '@/test/utils'
import PublishContent from './PublishContent'
import type { HeaderEntry } from './publish/HeadersEditor'
import type { OptionAvailability } from './publish/publishOptions'

interface TemplateLoadValues {
  name: string
  subject: string
  data: string
  headers: Record<string, string>
  wildcards: string[]
}

const hoisted = vi.hoisted(() => ({
  template: {
    name: 'Order created',
    subject: 'orders.created',
    data: '{"id":1}',
    headers: {} as Record<string, string>,
    wildcards: [] as string[],
  } as TemplateLoadValues,
}))

vi.mock('@/utils/toast', () => ({
  toast: { success: vi.fn(), error: vi.fn(), info: vi.fn(), warning: vi.fn() },
}))

vi.mock('@/api/publish', () => ({
  publishMessage: vi.fn().mockResolvedValue({ stream: 'ORDERS', sequence: 1 }),
  validateJSON: vi.fn().mockResolvedValue({ valid: true }),
}))

vi.mock('@/api/proto', () => ({
  getProtoMessageExample: vi.fn().mockResolvedValue({ example: {} }),
}))

vi.mock('@/api/messages', () => ({
  getMessages: vi.fn().mockResolvedValue({ messages: [] }),
}))

vi.mock('@/contexts/mappings', () => ({
  useSubjectMappingEntity: () => ({ mapping: null, messageType: null, sourceId: null }),
}))

vi.mock('@/contexts/proto', () => ({
  useProtoMessageEntity: () => ({ message: null, isLoading: false, error: null }),
}))

vi.mock('./publish/TemplateMenu', () => ({
  TemplateMenu: ({ onLoad }: { onLoad: (values: TemplateLoadValues) => void }) => (
    <button type="button" data-testid="load-template" onClick={() => onLoad(hoisted.template)}>
      Load template
    </button>
  ),
}))

vi.mock('@/components/common/editor/JsonCodeMirror', () => ({
  default: ({ value, onChange }: { value: string; onChange: (next: string) => void }) => (
    <textarea data-testid="json-editor" value={value} onChange={(e) => onChange(e.target.value)} />
  ),
}))

function Harness({
  initialPattern = 'orders.created',
  initialJson = '{}',
  initialHeaders = [],
  jetStreamOptions,
}: {
  initialPattern?: string
  initialJson?: string
  initialHeaders?: HeaderEntry[]
  jetStreamOptions?: OptionAvailability
}) {
  const [subjectPattern, setSubjectPattern] = useState(initialPattern)
  const [wildcardValues, setWildcardValues] = useState<string[]>([])
  const [messageJson, setMessageJson] = useState(initialJson)
  const [headers, setHeaders] = useState<HeaderEntry[]>(initialHeaders)

  return (
    <PublishContent
      subjects={['orders.created', 'orders.*.shipped']}
      connectionId="conn-1"
      streamName="ORDERS"
      subjectPattern={subjectPattern}
      onSubjectPatternChange={setSubjectPattern}
      wildcardValues={wildcardValues}
      onWildcardValuesChange={setWildcardValues}
      messageJson={messageJson}
      onMessageJsonChange={setMessageJson}
      headers={headers}
      onHeadersChange={setHeaders}
      jetStreamOptions={jetStreamOptions}
    />
  )
}

const publishButton = () => screen.getByRole('button', { name: /publish message/i })

function openJetStreamOptions() {
  fireEvent.click(screen.getByRole('button', { name: /JetStream options/ }))
}

async function renderPublish(props: Partial<Parameters<typeof Harness>[0]> = {}) {
  render(<Harness {...props} />)
  return screen.findByTestId('json-editor')
}

describe('PublishContent template loading', () => {
  beforeEach(() => {
    hoisted.template = {
      name: 'Order created',
      subject: 'orders.created',
      data: '{"id":1}',
      headers: {},
      wildcards: [],
    }
  })

  it('re-runs JSON validation when a template replaces invalid editor content', async () => {
    const editor = await renderPublish({ initialJson: '{}' })

    fireEvent.change(editor, { target: { value: '{ broken' } })
    expect(publishButton()).toBeDisabled()

    fireEvent.click(screen.getByTestId('load-template'))

    expect(screen.getByTestId('json-editor')).toHaveValue('{"id":1}')
    expect(publishButton()).toBeEnabled()
    expect(screen.queryByTestId('publish-disabled-reason')).toBeNull()
  })

  it('flags a template whose payload is not valid JSON, exactly as typing would', async () => {
    await renderPublish({ initialJson: '{"ok":true}' })

    hoisted.template = { ...hoisted.template, data: '{ nope' }
    fireEvent.click(screen.getByTestId('load-template'))

    expect(publishButton()).toBeDisabled()
    expect(screen.getByTestId('publish-disabled-reason')).toBeInTheDocument()
  })

  it('recomputes the payload size chip from the loaded template', async () => {
    await renderPublish({ initialJson: '{}' })

    hoisted.template = { ...hoisted.template, data: '{"id":1,"name":"abc"}' }
    fireEvent.click(screen.getByTestId('load-template'))

    expect(screen.getByTestId('payload-size')).toHaveTextContent('21 B')
  })

  it('validates a template that changes the subject pattern', async () => {
    const editor = await renderPublish({ initialJson: '{}' })

    fireEvent.change(editor, { target: { value: '{"a":1}' } })
    hoisted.template = {
      name: 'Shipped',
      subject: 'orders.*.shipped',
      data: '{ still broken',
      headers: {},
      wildcards: ['42'],
    }
    fireEvent.click(screen.getByTestId('load-template'))

    expect(publishButton()).toBeDisabled()
    expect(screen.getByTestId('publish-disabled-reason')).toBeInTheDocument()
  })

  it('explains why publishing is blocked when a template leaves wildcard slots empty', async () => {
    await renderPublish({ initialJson: '{}' })

    hoisted.template = {
      name: 'Shipped',
      subject: 'orders.*.shipped',
      data: '{"id":1}',
      headers: {},
      wildcards: [],
    }
    fireEvent.click(screen.getByTestId('load-template'))

    expect(publishButton()).toBeDisabled()
    expect(screen.getByTestId('publish-disabled-reason')).toHaveTextContent(/wildcard/i)
  })

  it('restores and revalidates the previous draft through the toast Undo action', async () => {
    const { toast } = await import('@/utils/toast')
    const editor = await renderPublish({ initialJson: '{}' })

    fireEvent.change(editor, { target: { value: '{ broken' } })
    fireEvent.click(screen.getByTestId('load-template'))
    expect(publishButton()).toBeEnabled()

    const calls = vi.mocked(toast.success).mock.calls
    const undo = calls[calls.length - 1]?.[1]?.action?.onClick
    act(() => undo?.())

    expect(screen.getByTestId('json-editor')).toHaveValue('{ broken')
    expect(publishButton()).toBeDisabled()
    expect(screen.getByTestId('publish-disabled-reason')).toBeInTheDocument()
  })
})

describe('PublishContent header name validation', () => {
  it('blocks publishing when a header key is not a valid header name', async () => {
    await renderPublish({ initialHeaders: [{ key: 'Bad Key', value: 'v' }] })

    expect(publishButton()).toBeDisabled()
    expect(screen.getByTestId('publish-disabled-reason')).toHaveTextContent(/header name/i)
    expect(screen.getByTestId('header-invalid-name')).toBeInTheDocument()
  })

  it('allows publishing once the invalid header key is fixed', async () => {
    await renderPublish({ initialHeaders: [{ key: 'X-Ok', value: 'v' }] })

    expect(screen.queryByTestId('header-invalid-name')).toBeNull()
    expect(publishButton()).toBeEnabled()
  })
})

describe('PublishContent JetStream options', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('publishes a counter increment without a body', async () => {
    const { publishMessage } = await import('@/api/publish')
    await renderPublish({ initialJson: '{"ignored":true}' })

    openJetStreamOptions()
    fireEvent.change(screen.getByLabelText('Counter increment'), { target: { value: '3' } })
    fireEvent.click(publishButton())

    await waitFor(() => expect(publishMessage).toHaveBeenCalled())
    expect(vi.mocked(publishMessage).mock.calls[0][0]).toEqual(
      expect.objectContaining({ data: null, headers: { 'Nats-Incr': '+3' }, message_type: undefined }),
    )
  })

  it('lets a counter increment publish with an empty body', async () => {
    const editor = await renderPublish({ initialJson: '{}' })

    fireEvent.change(editor, { target: { value: '' } })
    expect(publishButton()).toBeDisabled()
    openJetStreamOptions()
    fireEvent.change(screen.getByLabelText('Counter increment'), { target: { value: '+1' } })

    expect(publishButton()).toBeEnabled()
  })

  it('blocks publishing on an invalid option and says why', async () => {
    await renderPublish({ initialJson: '{}' })

    openJetStreamOptions()
    fireEvent.change(screen.getByLabelText('Message TTL'), { target: { value: 'soon' } })

    expect(publishButton()).toBeDisabled()
    expect(screen.getByTestId('publish-disabled-reason')).toHaveTextContent('TTL must be a duration like 30s, 5m or 1h')
  })

  it('adds the ttl header to a regular publish', async () => {
    const { publishMessage } = await import('@/api/publish')
    await renderPublish({ initialJson: '{"a":1}', initialHeaders: [{ key: 'X-Trace', value: 't' }] })

    openJetStreamOptions()
    fireEvent.change(screen.getByLabelText('Message TTL'), { target: { value: '5m' } })
    fireEvent.click(publishButton())

    await waitFor(() => expect(publishMessage).toHaveBeenCalled())
    expect(vi.mocked(publishMessage).mock.calls[0][0]).toEqual(
      expect.objectContaining({ data: { a: 1 }, headers: { 'X-Trace': 't', 'Nats-TTL': '5m' } }),
    )
  })

  it('reports the counter total after an increment', async () => {
    const { publishMessage } = await import('@/api/publish')
    const { toast } = await import('@/utils/toast')
    vi.mocked(publishMessage).mockResolvedValueOnce({ stream: 'HITS', sequence: 3, counter_value: '12' })
    await renderPublish({ initialJson: '{}' })

    openJetStreamOptions()
    fireEvent.change(screen.getByLabelText('Counter increment'), { target: { value: '+1' } })
    await act(async () => {
      fireEvent.click(publishButton())
    })

    expect(toast.success).toHaveBeenCalledWith('Message published to HITS, sequence: 3 · counter is now 12')
  })

  it('locks options the stream or server cannot take', async () => {
    render(<Harness jetStreamOptions={{ incrementUnavailable: 'Only counter streams accept increments' }} />)
    await screen.findByTestId('json-editor')

    openJetStreamOptions()
    expect(screen.getByLabelText('Counter increment')).toBeDisabled()
    expect(screen.getByText('Only counter streams accept increments')).toBeInTheDocument()
  })
})

describe('PublishContent increment loaded from history', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('republishes a hand-typed Nats-Incr header as a bodiless increment', async () => {
    const { publishMessage } = await import('@/api/publish')
    const editor = await renderPublish({ initialJson: '', initialHeaders: [{ key: 'Nats-Incr', value: '+5' }] })

    expect(editor).toHaveValue('')
    expect(publishButton()).toBeEnabled()
    fireEvent.click(publishButton())

    await waitFor(() => expect(publishMessage).toHaveBeenCalled())
    expect(vi.mocked(publishMessage).mock.calls[0][0]).toEqual(
      expect.objectContaining({ data: null, headers: { 'Nats-Incr': '+5' } }),
    )
  })

  it('does not treat a differently cased header as an increment', async () => {
    await renderPublish({ initialJson: '', initialHeaders: [{ key: 'nats-incr', value: '+5' }] })

    expect(publishButton()).toBeDisabled()
    expect(screen.getByTestId('publish-disabled-reason')).toHaveTextContent('Message body is empty')
  })
})
