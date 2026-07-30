import { useState } from 'react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, act } from '@/test/utils'
import PublishContent from './PublishContent'
import type { HeaderEntry } from './publish/HeadersEditor'

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

function Harness({ initialPattern = 'orders.created', initialJson = '{}' }) {
  const [subjectPattern, setSubjectPattern] = useState(initialPattern)
  const [wildcardValues, setWildcardValues] = useState<string[]>([])
  const [messageJson, setMessageJson] = useState(initialJson)
  const [headers, setHeaders] = useState<HeaderEntry[]>([])

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
    />
  )
}

const publishButton = () => screen.getByRole('button', { name: /publish message/i })

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
