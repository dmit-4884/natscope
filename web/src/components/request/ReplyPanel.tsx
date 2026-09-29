import { useEffect, useMemo, useState } from 'react'
import { decodeMessage } from '@/api/decode'
import { getErrorMessage } from '@/api/errors'
import type { RequestMessageRequest, RequestReply } from '@/contexts/messages'
import { useProtoMessageEntities } from '@/contexts/proto'
import { Alert, Badge, EmptyState, SearchableSelect, Spinner, SwitchHorizontalIcon } from '@/components/ui'
import { decodeBase64ToUtf8 } from '@/utils/base64'
import { formatBytes, formatNanoseconds } from '@/utils/formatters'
import PayloadViewer from '../messages/PayloadViewer'
import { formatTimeout, requestFailureKind } from './requestUtils'

const SERVICE_ERROR_HEADER = 'Nats-Service-Error'
const SERVICE_ERROR_CODE_HEADER = 'Nats-Service-Error-Code'
const NO_DECODING = ''

interface ReplyPanelProps {
  pending: boolean
  request: RequestMessageRequest | undefined
  reply: RequestReply | undefined
  error: unknown
  decodeAs: string
  onDecodeAsChange: (typeId: string) => void
}

interface DecodedReply {
  reply: RequestReply
  typeId: string
  data?: unknown
  error?: string
}

function parseJson(reply: RequestReply): unknown {
  try {
    const value: unknown = JSON.parse(decodeBase64ToUtf8(reply.data_base64))
    return typeof value === 'object' && value !== null ? value : undefined
  } catch {
    return undefined
  }
}

function RequestFailure({ error, request }: { error: unknown; request: RequestMessageRequest | undefined }) {
  const subject = request?.subject ?? ''
  switch (requestFailureKind(error)) {
    case 'no-responders':
      return (
        <Alert variant="warning" title="No responders" data-testid="reply-no-responders">
          Nothing is subscribed to <code>{subject}</code>. NATS answered right away instead of waiting for the timeout — check the
          subject or start the service.
        </Alert>
      )
    case 'timeout':
      return (
        <Alert
          variant="warning"
          title={`No reply within ${formatTimeout(request?.timeout_ms ?? 0)}`}
          data-testid="reply-timeout"
        >
          A responder may be subscribed to <code>{subject}</code> but did not answer in time. Raise the timeout or check the service.
        </Alert>
      )
    default:
      return (
        <Alert variant="error" title="Request failed" data-testid="reply-error">
          {getErrorMessage(error)}
        </Alert>
      )
  }
}

export function ReplyPanel({ pending, request, reply, error, decodeAs, onDecodeAsChange }: ReplyPanelProps) {
  const { messages } = useProtoMessageEntities()
  const [decoded, setDecoded] = useState<DecodedReply | null>(null)

  const typeOptions = useMemo(
    () => [
      { value: NO_DECODING, label: 'No decoding' },
      ...[...messages]
        .sort((a, b) => a.fullName.localeCompare(b.fullName))
        .map((m) => ({ value: m.id, label: m.fullName })),
    ],
    [messages],
  )
  const decodeType = messages.find((m) => m.id === decodeAs)

  useEffect(() => {
    if (!reply || !decodeType) return
    let cancelled = false
    const typeId = decodeType.id
    decodeMessage({ data_base64: reply.data_base64, message_type: decodeType.fullName, source_id: decodeType.sourceId })
      .then((result) => {
        if (cancelled) return
        setDecoded(
          result.success
            ? { reply, typeId, data: result.decoded }
            : { reply, typeId, error: result.error || 'Failed to decode the reply' },
        )
      })
      .catch((err: unknown) => {
        if (!cancelled) setDecoded({ reply, typeId, error: getErrorMessage(err) })
      })
    return () => {
      cancelled = true
    }
  }, [reply, decodeType])

  const jsonData = useMemo(() => (reply ? parseJson(reply) : undefined), [reply])
  const current = decoded && decoded.reply === reply && decoded.typeId === decodeType?.id ? decoded : null
  const isDecoding = !!reply && !!decodeType && !current
  const showDecoded = !!decodeType && !current?.error
  const serviceError = reply?.headers[SERVICE_ERROR_HEADER]
  const serviceErrorCode = reply?.headers[SERVICE_ERROR_CODE_HEADER]

  return (
    <div className="flex flex-col h-full min-h-0">
      <div className="px-4 py-3 border-b border-border flex items-center justify-between gap-3 flex-wrap">
        <div className="flex items-center gap-2 min-w-0">
          <h3 className="text-sm font-semibold text-content-primary">Reply</h3>
          {reply && !pending && (
            <>
              <Badge variant="success" size="sm">
                Received
              </Badge>
              <span className="text-xs text-content-tertiary tabular-nums" data-testid="reply-meta">
                {formatNanoseconds(Math.round(reply.duration_ms * 1_000_000))} · {formatBytes(reply.size)}
              </span>
            </>
          )}
        </div>
        {messages.length > 0 && (
          <div className="flex items-center gap-2">
            <span className="text-xs text-content-tertiary whitespace-nowrap" aria-hidden="true">
              Decode as
            </span>
            <div className="w-64 max-w-full">
              <SearchableSelect
                label="Decode reply as"
                options={typeOptions}
                value={decodeType ? decodeType.id : NO_DECODING}
                onChange={onDecodeAsChange}
                searchPlaceholder="Search proto types…"
              />
            </div>
          </div>
        )}
      </div>

      <div className="flex-1 min-h-0 flex flex-col" aria-live="polite" aria-busy={pending}>
        {pending ? (
          <div className="flex-1 flex flex-col items-center justify-center gap-2 text-sm text-content-tertiary" data-testid="reply-pending">
            <Spinner />
            <p>
              Waiting for a reply on <code>{request?.subject}</code>…
            </p>
            {request?.timeout_ms !== undefined && (
              <p className="text-xs text-content-muted">Gives up after {formatTimeout(request.timeout_ms)}</p>
            )}
          </div>
        ) : error ? (
          <div className="p-4">
            <RequestFailure error={error} request={request} />
          </div>
        ) : reply ? (
          <div className="flex-1 min-h-0 flex flex-col" data-testid="reply-success">
            <div className="px-4 pt-3 space-y-3">
              <p className="text-xs text-content-tertiary truncate" title={reply.subject}>
                Delivered on <code>{reply.subject}</code>
              </p>
              {serviceError && (
                <Alert
                  variant="error"
                  title={serviceErrorCode ? `Service error ${serviceErrorCode}` : 'Service error'}
                  data-testid="reply-service-error"
                >
                  {serviceError}
                </Alert>
              )}
              {current?.error && (
                <Alert variant="warning" title="Could not decode the reply">
                  {current.error}
                </Alert>
              )}
            </div>
            <div className="flex-1 min-h-0">
              <PayloadViewer
                rawData={reply.data_base64}
                decodedData={current?.data}
                jsonData={jsonData}
                headers={reply.headers}
                isDecoding={isDecoding}
                defaultMode={showDecoded ? 'decoded' : jsonData ? 'json' : 'raw'}
              />
            </div>
          </div>
        ) : (
          <EmptyState
            icon={<SwitchHorizontalIcon className="w-12 h-12" />}
            title="No reply yet"
            description="Send a request to see the first reply here. The request goes to every subscriber of the subject; the fastest answer wins."
          />
        )}
      </div>
    </div>
  )
}
