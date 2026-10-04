import { useMemo, useState } from 'react'
import { useOutletContext } from 'react-router-dom'
import { getErrorMessage } from '@/api/errors'
import { getProtoMessageExample } from '@/api/proto'
import { useMappingItems, useSubjectMappingEntity } from '@/contexts/mappings'
import { useRequestMessage } from '@/contexts/messages'
import { useMessageTypes, useTypeDescription } from '@/contexts/proto'
import { useRequestDraft, withRecentSubject } from '@/stores/requestDraftStore'
import { SubjectAutocomplete } from '@/components/common/SubjectAutocomplete'
import TemplateJsonEditor from '@/components/common/TemplateJsonEditor'
import { Select } from '@/components/ui'
import { jsonSyntaxError, processHelpers } from '@/utils/helpers'
import { toast } from '@/utils/toast'
import type { ConnectionOutletContext } from '../common/ConnectedLayout'
import { EncodingModeSelector, type EncodingMode } from '../streams/publish/EncodingModeSelector'
import { HeadersEditor, type HeaderEntry, type KnownHeaders } from '../streams/publish/HeadersEditor'
import { isValidHeaderName } from '../streams/publish/headerValidation'
import { PublishActionBar } from '../streams/publish/PublishActionBar'
import { TemplateMenu } from '../streams/publish/TemplateMenu'
import { buildSubject, hasWildcards } from '../streams/publish/subjectPatternUtils'
import { ReplyPanel } from './ReplyPanel'
import {
  REQUEST_TIMEOUT_OPTIONS_MS,
  formatTimeout,
  guessReplyType,
  literalSubjectError,
  literalSubjects,
} from './requestUtils'

const NO_KNOWN_HEADERS: KnownHeaders = {}
const SUBJECT_INPUT_ID = 'request-subject'
const SUBJECT_ERROR_ID = 'request-subject-error'
const TIMEOUT_SELECT_ID = 'request-timeout'
const TIMEOUT_HINT_ID = 'request-timeout-hint'

const TIMEOUT_OPTIONS = REQUEST_TIMEOUT_OPTIONS_MS.map((ms) => ({ value: String(ms), label: formatTimeout(ms) }))

export default function RequestPage() {
  const { connectionId, handleOpenMappings } = useOutletContext<ConnectionOutletContext>()
  const [draft, updateDraft] = useRequestDraft(connectionId)
  const [encodingMode, setEncodingMode] = useState<EncodingMode>('auto')
  const [exampleLoading, setExampleLoading] = useState(false)
  const request = useRequestMessage()

  const subject = draft.subject.trim()
  const subjectError = literalSubjectError(subject)

  const { data: mappings = [] } = useMappingItems()
  const subjectOptions = useMemo(
    () => [...new Set([...draft.recentSubjects, ...literalSubjects(mappings.map((m) => m.pattern))])],
    [draft.recentSubjects, mappings],
  )

  const {
    messageType: mappedMessageType,
    sourceId: mappedSourceId,
    framing: mappedFraming,
    pinnedFingerprint: mappedFingerprint,
  } = useSubjectMappingEntity(
    subject && !subjectError ? subject : null,
  )
  const serviceType = mappedMessageType ? undefined : draft.requestTypes[subject]
  const subjectType = mappedMessageType ?? serviceType?.messageType ?? null
  const subjectSource = mappedMessageType ? mappedSourceId : (serviceType?.sourceId ?? null)
  const isJsonMode = encodingMode === 'json' || (!subjectType && encodingMode !== 'proto')
  const messageType = isJsonMode ? undefined : subjectType ?? undefined
  const sourceId = isJsonMode ? undefined : subjectSource ?? undefined
  const { data: protoDescription, isLoading: protoLoading } = useTypeDescription(sourceId ?? null, messageType ?? null, true, messageType ? mappedFingerprint : undefined)
  const protoMessage = protoDescription?.messages[0]
  const { messages: protoTypes } = useMessageTypes()

  const payloadSyntaxError = useMemo(() => jsonSyntaxError(draft.payload), [draft.payload])
  const payloadIsEmpty = !draft.payload.trim()
  const payloadError = messageType
    ? payloadIsEmpty
      ? 'Enter the message as JSON — use {} for an empty message'
      : payloadSyntaxError
    : null
  const sendsPlainText = !messageType && !payloadIsEmpty && !!payloadSyntaxError
  const payloadBytes = useMemo(() => new TextEncoder().encode(draft.payload).length, [draft.payload])

  const hasInvalidHeaderName = draft.headers.some((h) => h.key.trim().length > 0 && !isValidHeaderName(h.key.trim()))

  const disabledReason = !subject
    ? 'Enter a subject'
    : subjectError
      ? 'Fix the subject above'
      : payloadError
        ? 'Fix the payload below'
        : hasInvalidHeaderName
          ? 'Fix the invalid header name below'
          : null
  const canSend = disabledReason === null

  const repliedSubject = request.variables?.subject ?? subject
  const storedReplyType = draft.replyTypes[repliedSubject]
  const decodeAs =
    storedReplyType ?? guessReplyType(request.variables?.message_type, request.variables?.source_id, protoTypes)

  const setHeaders = (headers: HeaderEntry[]) => updateDraft({ headers })

  const handleSend = () => {
    if (!canSend || request.isPending) return
    const shared: Record<string, string> = {}
    const headers = Object.fromEntries(
      draft.headers.filter((h) => h.key.trim()).map((h) => [h.key.trim(), processHelpers(h.value, shared)]),
    )
    updateDraft({ recentSubjects: withRecentSubject(draft.recentSubjects, subject) })
    request.mutate({
      connection_id: connectionId,
      subject: processHelpers(subject, shared),
      data: processHelpers(draft.payload, shared),
      headers,
      message_type: messageType,
      source_id: sourceId,
      schema_fingerprint: messageType ? mappedFingerprint : undefined,
      framing: messageType ? mappedFraming : undefined,
      timeout_ms: draft.timeoutMs,
    })
  }

  const handleContainerKeyDown = (e: React.KeyboardEvent) => {
    if ((e.ctrlKey || e.metaKey) && e.key === 'Enter' && !e.defaultPrevented) {
      e.preventDefault()
      handleSend()
    }
  }

  const handleUseExample = async () => {
    if (!subjectType || !subjectSource) return
    setExampleLoading(true)
    try {
      const response = await getProtoMessageExample(subjectSource, subjectType, mappedFingerprint)
      updateDraft({ payload: JSON.stringify(response.example, null, 2) })
    } catch (error) {
      toast.error(`Failed to generate example: ${getErrorMessage(error)}`)
    } finally {
      setExampleLoading(false)
    }
  }

  const handleLoadTemplate = ({
    name,
    subject: tplSubject,
    data,
    headers: tplHeaders,
    wildcards,
  }: {
    name: string
    subject: string
    data: string
    headers: Record<string, string>
    wildcards: string[]
  }) => {
    const prev = { subject: draft.subject, payload: draft.payload, headers: draft.headers.map((h) => ({ ...h })) }
    updateDraft({
      subject: hasWildcards(tplSubject) ? buildSubject(tplSubject, wildcards) : tplSubject,
      payload: data,
      headers: Object.entries(tplHeaders).map(([key, value]) => ({ key, value })),
    })
    toast.success(`Loaded template "${name}"`, {
      action: { label: 'Undo', onClick: () => updateDraft(prev) },
    })
  }

  const headersAsMap = useMemo(
    () => Object.fromEntries(draft.headers.filter((h) => h.key.trim()).map((h) => [h.key.trim(), h.value])),
    [draft.headers],
  )

  const completionSchema = useMemo(
    () => (messageType && protoDescription ? { messageType, description: protoDescription } : undefined),
    [messageType, protoDescription],
  )

  return (
    <div className="flex-1 flex flex-col overflow-hidden bg-surface-primary">
      <div className="px-6 pt-5 pb-4 border-b border-border">
        <h2 className="text-lg font-semibold text-content-primary">Request / Reply</h2>
        <p className="text-sm text-content-tertiary mt-0.5">
          Send a core NATS request to a service subject and inspect the first reply.
        </p>
      </div>

      <div className="flex-1 min-h-0 overflow-auto xl:overflow-hidden xl:grid xl:grid-cols-2">
        <section
          aria-label="Request"
          className="p-4 space-y-4 xl:overflow-auto xl:border-r border-border"
          onKeyDown={handleContainerKeyDown}
        >
          <div>
            <div className="flex items-center justify-between mb-2">
              <label htmlFor={SUBJECT_INPUT_ID} className="block text-sm font-medium text-gray-700">
                Subject
              </label>
              <TemplateMenu
                subjectPattern={subject}
                messageType={messageType ?? ''}
                messageJson={draft.payload}
                headers={headersAsMap}
                wildcards={[]}
                onLoad={handleLoadTemplate}
              />
            </div>
            <div className="flex items-start gap-3">
              <div className="flex-1 min-w-0">
                <SubjectAutocomplete
                  inputId={SUBJECT_INPUT_ID}
                  value={draft.subject}
                  onChange={(value) => updateDraft({ subject: value })}
                  options={subjectOptions}
                  placeholder="svc.echo"
                  invalid={!!subjectError}
                  describedBy={subjectError ? SUBJECT_ERROR_ID : undefined}
                />
                {subjectError && (
                  <p id={SUBJECT_ERROR_ID} className="mt-1 text-xs text-status-error-text" data-testid="request-subject-error">
                    {subjectError}
                  </p>
                )}
              </div>
              <div className="w-28 shrink-0">
                <label htmlFor={TIMEOUT_SELECT_ID} className="sr-only">
                  Timeout
                </label>
                <Select
                  id={TIMEOUT_SELECT_ID}
                  options={TIMEOUT_OPTIONS}
                  value={String(draft.timeoutMs)}
                  onChange={(e) => updateDraft({ timeoutMs: Number(e.target.value) })}
                  aria-describedby={TIMEOUT_HINT_ID}
                />
              </div>
            </div>
            <p id={TIMEOUT_HINT_ID} className="mt-1 text-xs text-content-muted">
              Waits up to {formatTimeout(draft.timeoutMs)} for the first reply.
            </p>
          </div>

          {subject && !subjectError && (
            <EncodingModeSelector
              encodingMode={encodingMode}
              isJsonMode={isJsonMode}
              messageType={messageType}
              mappedMessageType={subjectType}
              protoFieldCount={protoMessage?.fields.length}
              onModeChange={setEncodingMode}
              onAddMapping={() => handleOpenMappings(subject)}
            />
          )}
          {serviceType && !isJsonMode && (
            <p className="-mt-2 text-xs text-content-tertiary" data-testid="request-type-from-service">
              Type picked from the service endpoint.{' '}
              <button type="button" onClick={() => handleOpenMappings(subject)} className="text-accent hover:underline">
                Save it as a subject mapping
              </button>{' '}
              to keep it everywhere.
            </p>
          )}

          <div>
            <TemplateJsonEditor
              title="Payload"
              value={draft.payload}
              onChange={(payload) => updateDraft({ payload })}
              error={payloadError}
              placeholder={messageType ? '{"field": "value"}' : '{"field": "value"} or plain text — may be empty'}
              onUseExample={subjectType ? handleUseExample : undefined}
              exampleLoading={exampleLoading}
              exampleDisabled={protoLoading}
              sizeBytes={payloadBytes}
              schema={completionSchema}
              onSubmit={handleSend}
            />
            {sendsPlainText && (
              <p className="mt-2 text-xs text-content-muted" data-testid="request-plain-text-hint">
                Not JSON — the payload is sent as plain text.
              </p>
            )}
            {payloadIsEmpty && !messageType && (
              <p className="mt-2 text-xs text-content-muted" data-testid="request-empty-payload-hint">
                Empty payload — the request carries no body.
              </p>
            )}
          </div>

          <HeadersEditor
            headers={draft.headers}
            onAdd={() => setHeaders([...draft.headers, { key: '', value: '' }])}
            onRemove={(index) => setHeaders(draft.headers.filter((_, i) => i !== index))}
            onUpdate={(index, field, value) =>
              setHeaders(draft.headers.map((h, i) => (i === index ? { ...h, [field]: value } : h)))
            }
            knownHeaders={NO_KNOWN_HEADERS}
          />

          <PublishActionBar
            validationState="none"
            canPublish={canSend}
            disabledReason={disabledReason}
            isPublishing={request.isPending}
            onPublish={handleSend}
            submitLabel="Send Request"
            pendingLabel="Waiting for reply…"
            shortcutVerb="send"
          />
        </section>

        <section
          aria-label="Reply"
          className="min-h-[24rem] flex flex-col border-t border-border xl:border-t-0 xl:min-h-0 xl:overflow-hidden"
        >
          <ReplyPanel
            pending={request.isPending}
            request={request.variables}
            reply={request.data}
            error={request.error}
            decodeAs={decodeAs}
            onDecodeAsChange={(typeId) =>
              updateDraft({ replyTypes: { ...draft.replyTypes, [repliedSubject]: typeId } })
            }
          />
        </section>
      </div>
    </div>
  )
}
