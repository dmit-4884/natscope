import { useState, useEffect, useMemo, useRef, useCallback } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { CONNECTION_QUERY_PREFIX } from '@/hooks/useConnectionQuery'
import { toast } from '@/utils/toast'
import { publishMessage, validateJSON, type ValidationResult } from '@/api/publish'
import { getProtoMessageExample } from '@/api/proto'
import { getMessages } from '@/api/messages'
import { decodeBase64ToUtf8 } from '@/utils/base64'
import { useProtoMessageEntity } from '@/contexts/proto'
import { useSubjectMappingEntity } from '@/contexts/mappings'
import TemplateJsonEditor from '@/components/common/TemplateJsonEditor'
import { processHelpers } from '@/utils/helpers'
import { SubjectDropdown } from './publish/SubjectDropdown'
import { EditableSubject } from './publish/EditableSubject'
import { HeadersEditor, type HeaderEntry } from './publish/HeadersEditor'
import { EncodingModeSelector, type EncodingMode } from './publish/EncodingModeSelector'
import { ValidationResultDisplay } from './publish/ValidationResultDisplay'
import { TemplateMenu } from './publish/TemplateMenu'
import { PublishActionBar, type ValidationState } from './publish/PublishActionBar'
import { hasWildcards, parsePattern, buildSubject, countWildcards } from './publish/subjectPatternUtils'

interface PublishContentProps {
  subjects: string[]
  connectionId: string
  streamName: string
  subjectPattern: string
  onSubjectPatternChange: (pattern: string) => void
  wildcardValues: string[]
  onWildcardValuesChange: (values: string[]) => void
  messageJson: string
  onMessageJsonChange: (json: string) => void
  headers: HeaderEntry[]
  onHeadersChange: (headers: HeaderEntry[]) => void
  /** Stream max_msg_size in bytes (0/undefined = unlimited). */
  maxMsgSize?: number
  onOpenMappings?: (subjectPattern: string) => void
}

const AUTO_VALIDATE_DEBOUNCE_MS = 600

export default function PublishContent({
  subjects,
  connectionId,
  streamName,
  subjectPattern,
  onSubjectPatternChange,
  wildcardValues,
  onWildcardValuesChange,
  messageJson,
  onMessageJsonChange,
  headers,
  onHeadersChange,
  maxMsgSize,
  onOpenMappings,
}: PublishContentProps) {
  const [jsonError, setJsonError] = useState<string | null>(null)
  const [exampleLoading, setExampleLoading] = useState(false)
  const [prefillLoading, setPrefillLoading] = useState(false)
  const [validationResult, setValidationResult] = useState<ValidationResult | null>(null)
  const [validating, setValidating] = useState(false)
  const [encodingMode, setEncodingMode] = useState<EncodingMode>('auto')

  const addHeader = useCallback(
    () => onHeadersChange([...headers, { key: '', value: '' }]),
    [headers, onHeadersChange],
  )
  const removeHeader = useCallback(
    (index: number) => onHeadersChange(headers.filter((_, i) => i !== index)),
    [headers, onHeadersChange],
  )
  const updateHeader = useCallback(
    (index: number, field: 'key' | 'value', val: string) =>
      onHeadersChange(headers.map((h, i) => (i === index ? { ...h, [field]: val } : h))),
    [headers, onHeadersChange],
  )

  const queryClient = useQueryClient()
  const validateTimerRef = useRef<ReturnType<typeof setTimeout>>()

  const { messageType: mappedMessageType, sourceId: mappedSourceId } = useSubjectMappingEntity(
    subjectPattern || null,
  )
  const isJsonMode = encodingMode === 'json' || (!mappedMessageType && encodingMode !== 'proto')
  const messageType = isJsonMode ? undefined : mappedMessageType
  const sourceId = isJsonMode ? undefined : mappedSourceId ?? undefined

  const patternHasWildcards = useMemo(() => hasWildcards(subjectPattern), [subjectPattern])
  const patternSegments = useMemo(() => parsePattern(subjectPattern), [subjectPattern])
  const finalSubject = useMemo(
    () => (subjectPattern ? buildSubject(subjectPattern, wildcardValues) : ''),
    [subjectPattern, wildcardValues],
  )

  const payloadBytes = useMemo(() => new TextEncoder().encode(messageJson).length, [messageJson])
  const payloadOversize = !!maxMsgSize && maxMsgSize > 0 && payloadBytes > maxMsgSize

  const handlePatternChange = (pattern: string) => {
    onSubjectPatternChange(pattern)
    onWildcardValuesChange(new Array(countWildcards(pattern)).fill(''))
  }

  const handleWildcardChange = (index: number, value: string) => {
    const newValues = [...wildcardValues]
    newValues[index] = value
    onWildcardValuesChange(newValues)
  }

  const { message: protoMessage, isLoading: protoLoading } = useProtoMessageEntity(
    sourceId ?? null,
    messageType ?? null,
  )

  const publishMutation = useMutation({
    mutationFn: publishMessage,
    onSuccess: (response) => {
      toast.success(`Message published to ${response.stream}, sequence: ${response.sequence}`)
      if (response.duplicate) {
        toast.warning('Message was a duplicate (matched Nats-Msg-Id within the dedup window)')
      }
      // Invalidate every connection-scoped publishHistory cache entry.
      queryClient.invalidateQueries({
        predicate: (q) => {
          const k = q.queryKey
          return Array.isArray(k) && k[0] === CONNECTION_QUERY_PREFIX && k[2] === 'publishHistory'
        },
      })
    },
    onError: (error) => {
      toast.error(`Error: ${error instanceof Error ? error.message : 'Unknown error'}`)
      queryClient.invalidateQueries({
        predicate: (q) => {
          const k = q.queryKey
          return Array.isArray(k) && k[0] === CONNECTION_QUERY_PREFIX && k[2] === 'publishHistory'
        },
      })
    },
  })

  useEffect(() => {
    setJsonError(null)
    setValidationResult(null)
  }, [subjectPattern])

  const handleUseExample = async () => {
    if (!mappedMessageType || !mappedSourceId) return
    setExampleLoading(true)
    try {
      const response = await getProtoMessageExample(mappedSourceId, mappedMessageType)
      onMessageJsonChange(JSON.stringify(response.example, null, 2))
      setJsonError(null)
    } catch (error) {
      toast.error(`Failed to generate example: ${error instanceof Error ? error.message : 'Unknown error'}`)
    } finally {
      setExampleLoading(false)
    }
  }

  // Prefill the editor from the latest message on this subject (falls back to
  // raw pattern if wildcards unfilled).
  const handlePrefillFromLast = async () => {
    if (!subjectPattern) return
    setPrefillLoading(true)
    try {
      const allSlotsFilled = !patternHasWildcards || wildcardValues.every((v) => v.trim())
      const subjectFilter = allSlotsFilled ? finalSubject : subjectPattern
      const res = await getMessages(streamName, {
        connection_id: connectionId,
        subject_filter: subjectFilter,
        direction: 'backward',
        limit: 1,
        max_payload_bytes: 0,
      })
      const msg = res.messages[0]
      if (!msg) {
        toast.info(`No messages found for ${subjectFilter} yet`)
        return
      }
      let body: string
      if (msg.decoded && typeof msg.decoded === 'object') {
        body = JSON.stringify(msg.decoded, null, 2)
      } else {
        const raw = decodeBase64ToUtf8(msg.data_base64)
        try {
          body = JSON.stringify(JSON.parse(raw), null, 2)
        } catch {
          body = raw
        }
      }
      onMessageJsonChange(body)
      setJsonError(null)
      toast.success(`Loaded message #${msg.sequence} from ${msg.subject}`)
    } catch (error) {
      toast.error(`Failed to load last message: ${error instanceof Error ? error.message : 'Unknown error'}`)
    } finally {
      setPrefillLoading(false)
    }
  }

  const debouncedValidateJson = useCallback((value: string) => {
    if (validateTimerRef.current) clearTimeout(validateTimerRef.current)
    validateTimerRef.current = setTimeout(() => {
      try {
        const processed = processHelpers(value)
        JSON.parse(processed)
        setJsonError(null)
      } catch (e) {
        setJsonError(e instanceof Error ? e.message : 'Invalid JSON')
      }
    }, 200)
  }, [])

  useEffect(() => {
    return () => {
      if (validateTimerRef.current) clearTimeout(validateTimerRef.current)
    }
  }, [])

  const handleJsonChange = (value: string) => {
    onMessageJsonChange(value)
    if (!value.includes('{{')) {
      try {
        JSON.parse(value)
        setJsonError(null)
      } catch (e) {
        setJsonError(e instanceof Error ? e.message : 'Invalid JSON')
      }
    } else {
      debouncedValidateJson(value)
    }
  }

  // Debounced proto-schema auto-validation (proto mode); seq counter drops
  // stale responses.
  const validateSeqRef = useRef(0)
  useEffect(() => {
    if (!messageType || !sourceId || jsonError || !messageJson.trim()) {
      validateSeqRef.current++
      setValidating(false)
      setValidationResult(null)
      return
    }
    const seq = ++validateSeqRef.current
    const timer = setTimeout(async () => {
      // Flip the badge on only when the debounced call starts, so "Validating…"
      // doesn't flash on every keystroke.
      setValidating(true)
      try {
        const processedJson = processHelpers(messageJson)
        const data = JSON.parse(processedJson)
        const result = await validateJSON({ message_type: messageType, source_id: sourceId, data })
        if (validateSeqRef.current === seq) setValidationResult(result)
      } catch {
        if (validateSeqRef.current === seq) setValidationResult(null)
      } finally {
        if (validateSeqRef.current === seq) setValidating(false)
      }
    }, AUTO_VALIDATE_DEBOUNCE_MS)
    return () => clearTimeout(timer)
  }, [messageJson, messageType, sourceId, jsonError])

  const validationState: ValidationState = !messageType
    ? 'none'
    : validating
      ? 'validating'
      : validationResult
        ? validationResult.valid
          ? 'valid'
          : 'invalid'
        : 'none'

  const canPublish = useMemo(() => {
    if (!subjectPattern) return false
    if (patternHasWildcards && wildcardValues.some((v) => !v.trim())) return false
    if (jsonError) return false
    if (!messageJson.trim()) return false
    return true
  }, [subjectPattern, patternHasWildcards, wildcardValues, jsonError, messageJson])

  const handlePublish = () => {
    if (!canPublish) return
    try {
      const shared: Record<string, string> = {}
      const processedSubject = processHelpers(finalSubject, shared)
      const processedJson = processHelpers(messageJson, shared)
      const data = JSON.parse(processedJson)
      // Headers share the SAME `shared` map as subject/body so {{uuid}}
      // expands per-publish (else JetStream dedups it away).
      const headersMap = Object.fromEntries(
        headers.filter((h) => h.key.trim()).map((h) => [h.key.trim(), processHelpers(h.value, shared)]),
      )

      publishMutation.mutate({
        connection_id: connectionId,
        subject: processedSubject,
        subject_pattern: subjectPattern,
        message_type: messageType || undefined,
        source_id: sourceId || undefined,
        data,
        headers: Object.keys(headersMap).length > 0 ? headersMap : undefined,
      })
    } catch {
      toast.error('Invalid JSON format')
    }
  }

  // Tab-wide Cmd/Ctrl+Enter publish; defaultPrevented check avoids a double
  // publish (the JSON editor handles it first).
  const handleContainerKeyDown = (e: React.KeyboardEvent) => {
    if ((e.ctrlKey || e.metaKey) && e.key === 'Enter' && !e.defaultPrevented) {
      e.preventDefault()
      handlePublish()
    }
  }

  const handleAddMapping = () => {
    if (onOpenMappings) {
      onOpenMappings(subjectPattern)
    } else {
      toast.info('Add a mapping in the Mappings panel (Cmd+K → Mappings)')
    }
  }

  const handleLoadTemplate = ({
    name,
    subject,
    data,
    headers: tplHeaders,
    wildcards: tplWildcards,
  }: {
    name: string
    subject: string
    data: string
    headers: Record<string, string>
    wildcards: string[]
  }) => {
    // Snapshot current draft for the toast's Undo.
    const prev = {
      pattern: subjectPattern,
      wildcards: [...wildcardValues],
      messageJson,
      headers: headers.map((h) => ({ ...h })),
    }

    onSubjectPatternChange(subject)
    onMessageJsonChange(data)
    onHeadersChange(Object.entries(tplHeaders).map(([key, value]) => ({ key, value })))
    // Resize wildcard array to the new pattern's slots, fill positionally from
    // template; trailing slots stay empty.
    const slotCount = countWildcards(subject)
    const next = new Array(slotCount).fill('')
    for (let i = 0; i < Math.min(tplWildcards.length, slotCount); i++) next[i] = tplWildcards[i]
    onWildcardValuesChange(next)

    toast.success(`Loaded template "${name}"`, {
      action: {
        label: 'Undo',
        onClick: () => {
          onSubjectPatternChange(prev.pattern)
          onMessageJsonChange(prev.messageJson)
          onHeadersChange(prev.headers)
          onWildcardValuesChange(prev.wildcards)
        },
      },
    })
  }

  const headersAsMap = useMemo(() => {
    const out: Record<string, string> = {}
    for (const h of headers) {
      const k = h.key.trim()
      if (k) out[k] = h.value
    }
    return out
  }, [headers])

  // Schema-aware key completion in the editor (proto mode only).
  const completionFields = useMemo(() => {
    if (!messageType || !protoMessage) return undefined
    return protoMessage.fields.map((f) => ({
      name: f.name,
      type: f.type,
      repeated: f.isRepeated(),
      isMessage: f.isMessage,
    }))
  }, [messageType, protoMessage])

  return (
    <div className="flex-1 overflow-auto p-4 space-y-4" onKeyDown={handleContainerKeyDown}>
      <div>
        <div className="flex items-center justify-between mb-2">
          <label className="block text-sm font-medium text-gray-700">Subject Pattern</label>
          <TemplateMenu
            subjectPattern={subjectPattern}
            messageType={messageType || ''}
            messageJson={messageJson}
            headers={headersAsMap}
            wildcards={wildcardValues}
            onLoad={handleLoadTemplate}
          />
        </div>
        <SubjectDropdown
          value={subjectPattern}
          options={subjects}
          placeholder="Select Subject Pattern"
          onChange={handlePatternChange}
        />
      </div>

      {!subjectPattern && (
        <div className="bg-surface-secondary rounded-lg p-6 text-center">
          <p className="text-content-tertiary">Select a subject to publish a message</p>
        </div>
      )}

      {subjectPattern && (
        <EncodingModeSelector
          encodingMode={encodingMode}
          isJsonMode={isJsonMode}
          messageType={messageType}
          mappedMessageType={mappedMessageType}
          protoFieldCount={protoMessage?.fieldCount}
          onModeChange={setEncodingMode}
          onAddMapping={handleAddMapping}
        />
      )}

      {subjectPattern && (
        <>
          {patternHasWildcards && (
            <EditableSubject
              segments={patternSegments}
              wildcardValues={wildcardValues}
              onWildcardChange={handleWildcardChange}
            />
          )}

          <div className="flex-1 min-h-0">
            <TemplateJsonEditor
              value={messageJson}
              onChange={handleJsonChange}
              error={jsonError ?? undefined}
              onUseExample={handleUseExample}
              exampleLoading={exampleLoading}
              exampleDisabled={protoLoading || !mappedMessageType}
              onPrefillFromLast={handlePrefillFromLast}
              prefillLoading={prefillLoading}
              sizeBytes={payloadBytes}
              sizeLimitBytes={maxMsgSize}
              completionFields={completionFields}
              height="h-96"
              onSubmit={handlePublish}
            />
            {payloadOversize && (
              <div
                className="mt-2 px-3 py-2 bg-status-warning-bg border border-amber-200 rounded-lg text-xs text-amber-800"
                data-testid="payload-oversize-warning"
              >
                Payload is {payloadBytes.toLocaleString()} bytes — over this stream's max message size (
                {maxMsgSize!.toLocaleString()} bytes). NATS will reject the publish.
              </div>
            )}
            <ValidationResultDisplay result={validationResult} />
          </div>

          <HeadersEditor headers={headers} onAdd={addHeader} onRemove={removeHeader} onUpdate={updateHeader} />

          <PublishActionBar
            validationState={validationState}
            violationCount={validationResult?.violations?.length ?? 0}
            canPublish={canPublish}
            isPublishing={publishMutation.isPending}
            onPublish={handlePublish}
          />
        </>
      )}
    </div>
  )
}
