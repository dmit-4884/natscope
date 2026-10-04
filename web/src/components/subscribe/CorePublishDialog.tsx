import { useMemo, useState } from 'react'
import { getErrorMessage } from '@/api/errors'
import { useSubjectMappingEntity } from '@/contexts/mappings'
import { useCorePublish } from '@/contexts/messages'
import TemplateJsonEditor from '@/components/common/TemplateJsonEditor'
import { Button, Input, Modal } from '@/components/ui'
import ErrorAlert from '@/components/ui/ErrorAlert'
import { jsonSyntaxError, processHelpers } from '@/utils/helpers'
import { toast } from '@/utils/toast'
import { HeadersEditor, type HeaderEntry, type KnownHeaders } from '../streams/publish/HeadersEditor'
import { isValidHeaderName } from '../streams/publish/headerValidation'
import { publishSubjectError } from './subscribeUtils'

export interface CorePublishDraft {
  subject: string
  payload: string
  headers: HeaderEntry[]
}

interface Props {
  connectionId: string
  mode: 'resend' | 'reply'
  initial: CorePublishDraft
  onClose: () => void
}

const NO_KNOWN_HEADERS: KnownHeaders = {}
const SUBJECT_INPUT_ID = 'core-publish-subject'

export function CorePublishDialog({ connectionId, mode, initial, onClose }: Props) {
  const [subject, setSubject] = useState(initial.subject)
  const [payload, setPayload] = useState(initial.payload)
  const [headers, setHeaders] = useState<HeaderEntry[]>(initial.headers)
  const publish = useCorePublish()
  const isReply = mode === 'reply'

  const trimmed = subject.trim()
  const subjectError = publishSubjectError(subject)
  const { messageType, sourceId, framing, pinnedFingerprint } = useSubjectMappingEntity(subjectError ? null : trimmed)
  const payloadError = messageType
    ? payload.trim()
      ? jsonSyntaxError(payload)
      : 'Enter the message as JSON — use {} for an empty message'
    : null
  const hasInvalidHeaderName = headers.some((h) => h.key.trim().length > 0 && !isValidHeaderName(h.key.trim()))
  const payloadBytes = useMemo(() => new TextEncoder().encode(payload).length, [payload])
  const canSend = !subjectError && !payloadError && !hasInvalidHeaderName && !publish.isPending

  const handleSend = () => {
    if (!canSend) return
    const shared: Record<string, string> = {}
    publish.mutate(
      {
        connection_id: connectionId,
        subject: processHelpers(trimmed, shared),
        data: processHelpers(payload, shared),
        headers: Object.fromEntries(
          headers.filter((h) => h.key.trim()).map((h) => [h.key.trim(), processHelpers(h.value, shared)]),
        ),
        message_type: messageType ?? undefined,
        source_id: messageType ? (sourceId ?? undefined) : undefined,
        schema_fingerprint: messageType ? pinnedFingerprint : undefined,
        framing: messageType ? framing : undefined,
      },
      {
        onSuccess: () => {
          toast.success(isReply ? 'Reply sent' : `Published to ${trimmed}`)
          onClose()
        },
      },
    )
  }

  return (
    <Modal
      isOpen
      onClose={onClose}
      title={isReply ? 'Reply to request' : 'Resend message'}
      size="lg"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button disabled={!canSend} loading={publish.isPending} onClick={handleSend}>
            {isReply ? 'Send reply' : 'Send'}
          </Button>
        </>
      }
    >
      <div
        className="px-6 py-4 space-y-4 overflow-y-auto"
        onKeyDown={(e) => {
          if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
            e.preventDefault()
            handleSend()
          }
        }}
      >
        <div>
          <label htmlFor={SUBJECT_INPUT_ID} className="block text-sm font-medium text-gray-700 mb-1">
            {isReply ? 'Reply subject' : 'Subject'}
          </label>
          <Input
            id={SUBJECT_INPUT_ID}
            mono
            value={subject}
            readOnly={isReply}
            onChange={(e) => setSubject(e.target.value)}
            error={!!subjectError && !isReply}
            errorMessage={isReply ? undefined : (subjectError ?? undefined)}
          />
          <p className="mt-1 text-xs text-content-tertiary">
            {isReply
              ? 'The requester waits on this inbox only until its timeout, so send the reply soon.'
              : 'Published over core NATS: subscribers get it now, nothing stores it.'}
          </p>
        </div>

        <div>
          <TemplateJsonEditor
            title="Payload"
            value={payload}
            onChange={setPayload}
            error={payloadError}
            placeholder={messageType ? '{"field": "value"}' : '{"field": "value"} or plain text — may be empty'}
            sizeBytes={payloadBytes}
            height="12rem"
            onSubmit={handleSend}
          />
          <p className="mt-1 text-xs text-content-tertiary">
            {messageType ? (
              <>
                Encoded as Protobuf <code>{messageType}</code> through the subject mapping.
              </>
            ) : (
              'Sent as typed: JSON or plain text.'
            )}
          </p>
        </div>

        <HeadersEditor
          headers={headers}
          onAdd={() => setHeaders([...headers, { key: '', value: '' }])}
          onRemove={(index) => setHeaders(headers.filter((_, i) => i !== index))}
          onUpdate={(index, field, value) => setHeaders(headers.map((h, i) => (i === index ? { ...h, [field]: value } : h)))}
          knownHeaders={NO_KNOWN_HEADERS}
        />

        {publish.error && <ErrorAlert message={getErrorMessage(publish.error)} />}
      </div>
    </Modal>
  )
}
