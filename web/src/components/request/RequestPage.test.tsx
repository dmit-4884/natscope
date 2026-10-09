import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useState } from 'react'
import { Code, ConnectError } from '@connectrpc/connect'
import { create, toBinary } from '@bufbuild/protobuf'
import { ErrorInfoSchema } from '@/gen/google/rpc/error_details_pb'
import { render, screen, fireEvent, waitFor } from '@/test/utils'
import { requestMessage, type RequestReply } from '@/api/publish'
import { clearAllRequestDrafts, patchRequestDraft } from '@/stores/requestDraftStore'
import { encodeBytesToBase64 } from '@/utils/base64'
import RequestPage from './RequestPage'

vi.mock('react-router-dom', async (importOriginal) => ({
  ...(await importOriginal<typeof import('react-router-dom')>()),
  useOutletContext: () => ({ connectionId: 'conn-1', handleOpenMappings: vi.fn() }),
}))

vi.mock('@/utils/toast', () => ({
  toast: { success: vi.fn(), error: vi.fn(), info: vi.fn(), warning: vi.fn() },
}))

vi.mock('@/api/publish', () => ({
  requestMessage: vi.fn(),
}))

vi.mock('@/api/decode', () => ({
  decodeMessage: vi.fn(),
}))

vi.mock('@/api/proto', () => ({
  getProtoMessageExample: vi.fn().mockResolvedValue({ example: {} }),
}))

vi.mock('@/contexts/mappings', () => ({
  useMappingItems: () => ({ data: [] }),
  useSubjectMappingEntity: () => ({ mapping: null, messageType: null, sourceId: null }),
}))

vi.mock('@/contexts/proto', () => ({
  useTypeDescription: () => ({ data: undefined, isLoading: false }),
  useMessageTypes: () => ({ messages: [], isLoading: false }),
}))

vi.mock('../streams/publish/TemplateMenu', () => ({
  TemplateMenu: () => null,
}))

vi.mock('@/components/common/editor/JsonCodeMirror', () => ({
  default: ({ value, onChange }: { value: string; onChange: (next: string) => void }) => (
    <textarea data-testid="json-editor" value={value} onChange={(e) => onChange(e.target.value)} />
  ),
}))

const mockedRequest = vi.mocked(requestMessage)

function reply(body: string, headers: Record<string, string> = {}): RequestReply {
  const bytes = new TextEncoder().encode(body)
  return {
    subject: '_INBOX.abc.1',
    data_base64: encodeBytesToBase64(bytes),
    size: bytes.length,
    headers,
    duration_ms: 1.5,
  }
}

function reasonError(code: Code, reason: string): ConnectError {
  const err = new ConnectError('failed', code)
  err.details = [{ type: ErrorInfoSchema.typeName, value: toBinary(ErrorInfoSchema, create(ErrorInfoSchema, { reason })) }]
  return err
}

const sendButton = () => screen.getByRole('button', { name: /send request/i })

async function renderPage() {
  render(<RequestPage />)
  return screen.findByTestId('json-editor')
}

describe('RequestPage', () => {
  beforeEach(() => {
    clearAllRequestDrafts()
    mockedRequest.mockReset()
  })

  it('sends a plain-text request and shows the reply', async () => {
    mockedRequest.mockResolvedValue(reply('{"pong":true}'))
    const editor = await renderPage()

    fireEvent.change(screen.getByLabelText('Subject'), { target: { value: 'svc.echo' } })
    fireEvent.change(editor, { target: { value: 'ping' } })
    expect(screen.getByTestId('request-plain-text-hint')).toBeInTheDocument()

    fireEvent.click(sendButton())

    await screen.findByTestId('reply-success')
    expect(mockedRequest.mock.calls[0][0]).toEqual({
      connection_id: 'conn-1',
      subject: 'svc.echo',
      data: 'ping',
      headers: {},
      message_type: undefined,
      source_id: undefined,
      timeout_ms: 5000,
    })
    expect(screen.getByText('Received')).toBeInTheDocument()
    expect(screen.getByText('_INBOX.abc.1')).toBeInTheDocument()
  })

  it('still shows the last reply after a visit to another page', async () => {
    mockedRequest.mockResolvedValue(reply('{"pong":true}'))
    function AwayAndBack() {
      const [here, setHere] = useState(true)
      return (
        <>
          <button type="button" onClick={() => setHere((v) => !v)}>
            Toggle page
          </button>
          {here ? <RequestPage /> : <p>Another page</p>}
        </>
      )
    }
    render(<AwayAndBack />)
    fireEvent.change(await screen.findByLabelText('Subject'), { target: { value: 'svc.echo' } })
    fireEvent.click(sendButton())
    await screen.findByTestId('reply-success')

    fireEvent.click(screen.getByRole('button', { name: 'Toggle page' }))
    fireEvent.click(screen.getByRole('button', { name: 'Toggle page' }))

    expect(await screen.findByTestId('reply-success')).toBeInTheDocument()
  })

  it('brings a new reply into view when the reply sits below the request', async () => {
    const original = Element.prototype.scrollIntoView
    const scrollIntoView = vi.fn()
    Element.prototype.scrollIntoView = scrollIntoView
    mockedRequest.mockResolvedValue(reply('{"pong":true}'))
    await renderPage()
    fireEvent.change(screen.getByLabelText('Subject'), { target: { value: 'svc.echo' } })

    fireEvent.click(sendButton())
    await screen.findByTestId('reply-success')

    expect(scrollIntoView.mock.instances).toContain(screen.getByRole('region', { name: 'Reply' }))
    Element.prototype.scrollIntoView = original
  })

  it('sends with the chosen timeout', async () => {
    mockedRequest.mockResolvedValue(reply(''))
    await renderPage()

    fireEvent.change(screen.getByLabelText('Subject'), { target: { value: 'svc.echo' } })
    fireEvent.change(screen.getByLabelText('Timeout'), { target: { value: '500' } })
    fireEvent.click(sendButton())

    await waitFor(() => expect(mockedRequest).toHaveBeenCalled())
    expect(mockedRequest.mock.calls[0][0]).toEqual(expect.objectContaining({ timeout_ms: 500 }))
  })

  it('explains that nothing listens on the subject', async () => {
    mockedRequest.mockRejectedValue(reasonError(Code.Unavailable, 'NATS_NO_RESPONDERS'))
    await renderPage()

    fireEvent.change(screen.getByLabelText('Subject'), { target: { value: 'svc.nobody' } })
    fireEvent.click(sendButton())

    expect(await screen.findByTestId('reply-no-responders')).toHaveTextContent('svc.nobody')
  })

  it('names a refused permission calmly', async () => {
    const err = new ConnectError('no permission to publish to "calc.div"', Code.PermissionDenied)
    err.details = [
      {
        type: ErrorInfoSchema.typeName,
        value: toBinary(
          ErrorInfoSchema,
          create(ErrorInfoSchema, { reason: 'NATS_PERMISSION_VIOLATION', metadata: { operation: 'publish', subject: 'calc.div' } }),
        ),
      },
    ]
    mockedRequest.mockRejectedValue(err)
    await renderPage()

    fireEvent.change(screen.getByLabelText('Subject'), { target: { value: 'calc.div' } })
    fireEvent.click(sendButton())

    expect(await screen.findByTestId('access-denied-notice')).toHaveTextContent('publish to calc.div')
    expect(screen.queryByTestId('reply-error')).not.toBeInTheDocument()
  })

  it('reports a timeout with the configured wait', async () => {
    mockedRequest.mockRejectedValue(reasonError(Code.DeadlineExceeded, 'NATS_TIMEOUT'))
    await renderPage()

    fireEvent.change(screen.getByLabelText('Subject'), { target: { value: 'svc.slow' } })
    fireEvent.click(sendButton())

    expect(await screen.findByTestId('reply-timeout')).toHaveTextContent('No reply within 5 s')
  })

  it('surfaces a NATS micro service error from the reply headers', async () => {
    mockedRequest.mockResolvedValue(
      reply('', { 'Nats-Service-Error': 'bad input', 'Nats-Service-Error-Code': '400' }),
    )
    await renderPage()

    fireEvent.change(screen.getByLabelText('Subject'), { target: { value: 'svc.add' } })
    fireEvent.click(sendButton())

    const alert = await screen.findByTestId('reply-service-error')
    expect(alert).toHaveTextContent('Service error 400')
    expect(alert).toHaveTextContent('bad input')
  })

  it('refuses a wildcard subject', async () => {
    await renderPage()

    fireEvent.change(screen.getByLabelText('Subject'), { target: { value: 'svc.*' } })

    expect(screen.getByTestId('request-subject-error')).toHaveTextContent('wildcards')
    expect(sendButton()).toBeDisabled()
    expect(screen.getByLabelText('Subject')).toHaveAttribute('aria-invalid', 'true')
  })

  it('requires a subject before sending', async () => {
    await renderPage()

    expect(sendButton()).toBeDisabled()
    expect(screen.getByTestId('publish-disabled-reason')).toHaveTextContent('Enter a subject')
  })

  it('encodes with the request type a service endpoint picked when no mapping exists', async () => {
    mockedRequest.mockResolvedValue(reply(''))
    patchRequestDraft('conn-1', {
      subject: 'orders.create',
      payload: '{"id":1}',
      requestTypes: { 'orders.create': { messageType: 'shop.CreateOrderRequest', sourceId: 'src-1' } },
    })
    await renderPage()

    expect(screen.getByTestId('request-type-from-service')).toBeInTheDocument()
    fireEvent.click(sendButton())

    await waitFor(() => expect(mockedRequest).toHaveBeenCalled())
    expect(mockedRequest.mock.calls[0][0]).toEqual(
      expect.objectContaining({ subject: 'orders.create', message_type: 'shop.CreateOrderRequest', source_id: 'src-1' }),
    )
  })
})
