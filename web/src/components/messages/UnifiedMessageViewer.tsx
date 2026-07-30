import { useState, useEffect, useCallback, useMemo, useRef } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { toast } from '@/utils/toast'
import { copyText } from '@/utils/clipboard'
import { CONNECTION_QUERY_PREFIX } from '@/hooks/useConnectionQuery'
import { useStreamDetail } from '@/contexts/streams'
import { useMappingItems, type MappingItem } from '@/contexts/mappings'
import ErrorAlert from '@/components/ui/ErrorAlert'
import { RefreshIcon, TrashIcon, PlusIcon, ChevronDownIcon, ChevronUpIcon, DocumentIcon } from '@/components/ui'
import Tooltip from '@/components/common/Tooltip'
import { decodeMessage } from '@/api/decode'
import { getMessage } from '@/api/messages'
import { deleteMessage } from '@/api/management'
import { getSubjectPattern as getPatternFromSubject, matchesPattern as subjectMatchesPattern } from '@/contexts/messages'
import { decodeBase64ToUtf8 } from '@/utils/base64'
import { getErrorMessage } from '@/api/errors'
import { useDisplayPreferences, useConfirmation, useBehaviorPolicy } from '@/contexts/settings'
import PayloadViewer from './PayloadViewer'
import { BookmarkButton } from './Bookmarks'
import { MessageDeleteDialog } from './MessageDeleteDialog'
import { buildResendDraft, type ResendDraft } from './resend'
import type { MessageNavigation } from './unified/useMessageNavigation'
import type { SelectedMessage } from './UnifiedMessageList'

interface UnifiedMessageViewerProps {
  streamName: string | null
  connectionId: string | null
  selectedMessage: SelectedMessage | null
  onOpenMappings?: (subjectPattern: string) => void
  /**
   * Called after a message is deleted so the parent can clear the selection.
   */
  onDeleted?: () => void
  /**
   * Called to load a message into the publish form and switch to the Publish
   * tab.
   */
  onResend?: (draft: ResendDraft) => void
  /** Prev/next stepping controls; buttons render only when provided. */
  navigation?: MessageNavigation
}

// Parse base64 as JSON (content_type 'json' only).
const parseJsonData = (base64Data: string): unknown | null => {
  try {
    return JSON.parse(decodeBase64ToUtf8(base64Data))
  } catch {
    return null
  }
}

// Pattern specificity (higher = more specific).
const patternSpecificity = (pattern: string): number => {
  const parts = pattern.split('.')
  let specificity = parts.length * 10
  for (const p of parts) {
    if (p === '>') specificity -= 5
    if (p === '*') specificity -= 2
  }
  return specificity
}

const findMappingMatch = (subject: string, mappings: MappingItem[]): MappingItem | null => {
  const exact = mappings.find((m) => m.pattern === subject)
  if (exact) return exact

  let bestMatch: MappingItem | null = null
  let bestSpecificity = -Infinity
  for (const m of mappings) {
    if (subjectMatchesPattern(subject, m.pattern)) {
      const specificity = patternSpecificity(m.pattern)
      if (!bestMatch || specificity > bestSpecificity) {
        bestMatch = m
        bestSpecificity = specificity
      }
    }
  }
  return bestMatch
}

export default function UnifiedMessageViewer({
  streamName,
  connectionId,
  selectedMessage,
  onOpenMappings,
  onDeleted,
  onResend,
  navigation,
}: UnifiedMessageViewerProps) {
  const display = useDisplayPreferences()
  const queryClient = useQueryClient()
  const deleteConfirmation = useConfirmation('deleteMessage')
  const behavior = useBehaviorPolicy()
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [selectedProtoType, setSelectedProtoType] = useState('')
  const [selectedSourceId, setSelectedSourceId] = useState('')
  const [decodedData, setDecodedData] = useState<unknown>(null)
  const [decoding, setDecoding] = useState(false)
  const [decodeError, setDecodeError] = useState<string | null>(null)
  const [hasDecodedForType, setHasDecodedForType] = useState<string | null>(null)

  const { data: mappings = [] } = useMappingItems()

  // "Load full payload" override; reset on selection change so the button
  // reappears.
  const [fullMessage, setFullMessage] = useState<SelectedMessage | null>(null)
  const [loadingFull, setLoadingFull] = useState(false)
  const [loadFullError, setLoadFullError] = useState<string | null>(null)

  const selectionIdRef = useRef<string | undefined>(undefined)
  useEffect(() => {
    selectionIdRef.current = selectedMessage?.id
  }, [selectedMessage?.id])

  useEffect(() => {
    setFullMessage(null)
    setLoadFullError(null)
    setLoadingFull(false)
  }, [selectedMessage?.id])

  // selectedMessage already carries all data; list endpoint returns identical
  // data, no re-fetch.
  const message = (fullMessage ?? selectedMessage) as SelectedMessage | null
  const isTruncated = (selectedMessage?.truncated && !fullMessage) === true

  const jsonData = useMemo(
    () =>
      !decodedData && message?.data_base64 && message?.content_type === 'json'
        ? parseJsonData(message.data_base64)
        : null,
    [message?.data_base64, message?.content_type, decodedData],
  )

  const handleLoadFull = useCallback(async () => {
    if (!connectionId || !streamName || !selectedMessage?.sequence) return
    const idAtCall = selectedMessage.id
    setLoadingFull(true)
    setLoadFullError(null)
    try {
      const full = await getMessage(connectionId, streamName, selectedMessage.sequence)
      if (selectionIdRef.current !== idAtCall) return
      // Normalize snake_case API fields to camelCase; else merged message
      // exposes undefined decodedType and decode silently nulls.
      setFullMessage({
        ...full,
        decodedType: full.decoded_type,
        decodeError: full.decode_error,
      } as SelectedMessage)
      toast.success('Full payload loaded')
    } catch (err: unknown) {
      if (selectionIdRef.current !== idAtCall) return
      const msg = getErrorMessage(err)
      setLoadFullError(msg)
      toast.error(`Failed to load: ${msg}`)
    } finally {
      if (selectionIdRef.current === idAtCall) setLoadingFull(false)
    }
  }, [connectionId, streamName, selectedMessage?.sequence, selectedMessage?.id])

  const { data: streamDetail } = useStreamDetail(streamName, connectionId)

  // Delete viewed message (by sequence); invalidates list + stats, then clears
  // selection.
  const performDelete = useCallback(
    async (secure: boolean) => {
      if (!connectionId || !streamName || !selectedMessage?.sequence) return
      setDeleting(true)
      try {
        await deleteMessage(connectionId, streamName, selectedMessage.sequence, secure)
        await queryClient.invalidateQueries({
          queryKey: [CONNECTION_QUERY_PREFIX, connectionId, 'messages', streamName],
        })
        queryClient.invalidateQueries({
          queryKey: [CONNECTION_QUERY_PREFIX, connectionId, 'streamStats', streamName],
        })
        // Refresh stream detail (key mirrors useStreamDetail) —
        // totalCount/first_seq move on delete, else count header goes stale.
        queryClient.invalidateQueries({
          queryKey: [CONNECTION_QUERY_PREFIX, connectionId, 'stream', streamName],
        })
        toast.success(`Message #${selectedMessage.sequence} deleted`)
        setDeleteOpen(false)
        onDeleted?.()
      } catch (err) {
        toast.error(`Failed to delete: ${getErrorMessage(err)}`)
      } finally {
        setDeleting(false)
      }
    },
    [connectionId, streamName, selectedMessage?.sequence, queryClient, onDeleted],
  )

  // Open confirm dialog, or delete straight away when confirmation is disabled.
  const requestDelete = useCallback(() => {
    if (!deleteConfirmation.enabled) {
      void performDelete(behavior.secureDeleteDefault)
      return
    }
    setDeleteOpen(true)
  }, [deleteConfirmation.enabled, behavior.secureDeleteDefault, performDelete])

  // Prefer decoded view so the user edits JSON not base64. CRITICAL: list
  // payloads are truncated — must fetch FULL before resend or bytes corrupt.
  const handleResend = useCallback(async () => {
    if (!message?.subject || !onResend) return

    // Resolve to full, untruncated message before building the draft.
    let resendMessage = message
    let resendDecoded =
      decodedData ?? jsonData ?? (typeof message.decoded === 'object' ? message.decoded : undefined)
    if (isTruncated && connectionId && streamName && selectedMessage?.sequence) {
      try {
        const full = await getMessage(connectionId, streamName, selectedMessage.sequence)
        // Normalize snake_case→camelCase (as handleLoadFull) so cached
        // selection updates and merged decoded view stays consistent.
        const normalized = {
          ...full,
          decodedType: full.decoded_type,
          decodeError: full.decode_error,
        } as SelectedMessage
        setFullMessage(normalized)
        resendMessage = normalized
        resendDecoded =
          typeof normalized.decoded === 'object' ? normalized.decoded : undefined
      } catch (err: unknown) {
        toast.error(`Failed to load full payload: ${getErrorMessage(err)}`)
        return
      }
    }

    const subjects = streamDetail?.subjects ?? []
    // Resolve subject to a configured stream pattern (first match), else
    // inferred — mirrors getSubjectPattern below.
    const resolvePattern = (subject: string): string => {
      for (const p of subjects) {
        if (subjectMatchesPattern(subject, p)) return p
      }
      return getPatternFromSubject(subject)
    }
    const draft = buildResendDraft(
      {
        subject: resendMessage.subject,
        dataBase64: resendMessage.data_base64,
        decoded: resendDecoded,
        headers: resendMessage.headers,
      },
      resolvePattern,
    )
    // Binary (non-UTF-8) payload with no decoded form can't be edited as text —
    // abort vs loading replacement-char garbage.
    if (draft.binaryUndecodable) {
      toast.error('Cannot edit & resend a binary payload — no text or decoded representation available')
      return
    }
    onResend(draft)
    toast.success('Loaded into publish form')
  }, [
    message,
    decodedData,
    jsonData,
    onResend,
    streamDetail?.subjects,
    isTruncated,
    connectionId,
    streamName,
    selectedMessage?.sequence,
  ])

  // Subject pattern from stream config, else inferred.
  const getSubjectPattern = (subject: string): string => {
    if (!streamDetail?.subjects || streamDetail.subjects.length === 0) {
      return getPatternFromSubject(subject)
    }

    for (const pattern of streamDetail.subjects) {
      if (subjectMatchesPattern(subject, pattern)) {
        return pattern
      }
    }

    return getPatternFromSubject(subject)
  }

  // Reset decode state on message change, applying server-decoded data if
  // present.
  useEffect(() => {
    if (message?.decoded && message?.decodedType) {
      setDecodedData(message.decoded)
      setSelectedProtoType(message.decodedType)
      setHasDecodedForType(message.decodedType)
    } else {
      setDecodedData(null)
      setSelectedProtoType('')
      setSelectedSourceId('')
      setHasDecodedForType(null)
    }
    setDecodeError(null)
    setDecoding(false)
  }, [selectedMessage?.id, message?.decoded, message?.decodedType])

  // Auto-load saved proto type + source for this subject by pattern matching;
  // skip if server already decoded.
  useEffect(() => {
    if (message?.subject && !selectedMessage?.isLive && !message?.decodedType) {
      const match = findMappingMatch(message.subject, mappings)
      if (match) {
        setSelectedProtoType(match.messageType)
        setSelectedSourceId(match.sourceId)
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [message?.subject, selectedMessage?.id, mappings, message?.decodedType])

  // Auto-decode via gRPC Codec only when server did NOT decode.
  useEffect(() => {
    // Hard guard: skip if server already decoded.
    if (message?.decoded && message?.decodedType) return

    if (
      selectedProtoType &&
      selectedSourceId &&
      message?.data_base64 &&
      !decoding &&
      hasDecodedForType !== selectedProtoType
    ) {
      const currentProtoType = selectedProtoType
      const currentSourceId = selectedSourceId
      const idAtCall = selectedMessage?.id

      const performDecode = async () => {
        setDecoding(true)
        setDecodeError(null)
        setDecodedData(null)

        try {
          const result = await decodeMessage({
            data_base64: message.data_base64,
            message_type: currentProtoType,
            source_id: currentSourceId,
          })
          if (selectionIdRef.current !== idAtCall) return

          if (result.success && result.decoded) {
            setDecodedData(result.decoded)
            setHasDecodedForType(currentProtoType)
          } else {
            setDecodeError(result.error || 'Failed to decode message')
            setHasDecodedForType(null)
          }
        } catch (err) {
          if (selectionIdRef.current !== idAtCall) return
          setDecodeError(`Decode failed: ${(err as Error).message}`)
          setHasDecodedForType(null)
        } finally {
          if (selectionIdRef.current === idAtCall) setDecoding(false)
        }
      }

      performDecode()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedProtoType, selectedSourceId, message?.data_base64, hasDecodedForType, selectedMessage?.isLive])

  const willAutoDecode = useMemo(() => {
    if (message?.decoded && message?.decodedType) return false
    if (!message?.data_base64) return false
    if (selectedProtoType && selectedSourceId) return true
    return !selectedMessage?.isLive && !!message?.subject && findMappingMatch(message.subject, mappings) != null
  }, [
    message?.decoded,
    message?.decodedType,
    message?.data_base64,
    message?.subject,
    selectedProtoType,
    selectedSourceId,
    mappings,
    selectedMessage?.isLive,
  ])

  if (!selectedMessage) {
    return (
      <div className="h-full flex items-center justify-center text-content-tertiary">
        <div className="text-center">
          <DocumentIcon className="mx-auto h-12 w-12 text-content-muted mb-4" />
          <p className="text-sm">Select a message to view details</p>
        </div>
      </div>
    )
  }

  const displayMessage = message

  // Divider between the nav pair and the action icons — only when both sides render.
  const showActionSeparator =
    !!navigation &&
    !!displayMessage &&
    ((!!streamName && !!displayMessage.sequence) || (!!onResend && !!displayMessage.subject))

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="p-4 bg-surface-primary border-b">
        <div className="flex items-center justify-between gap-3 mb-2">
          <div className="flex items-center gap-2 min-w-0">
            <h2 className="text-lg font-semibold text-content-primary whitespace-nowrap">
              {displayMessage?.sequence ? `Message #${displayMessage.sequence}` : 'Live Message'}
            </h2>
            {displayMessage?.isLive && (
              <span className="px-2 py-0.5 text-xs font-medium bg-status-success-light text-green-700 rounded-full">
                Live
              </span>
            )}
          </div>
          <div className="flex items-center gap-0.5 flex-none">
            {navigation && (
              <span
                className="inline-flex rounded-lg border border-border overflow-hidden"
                role="group"
                aria-label="Message navigation"
              >
                <Tooltip content={navigation.canNavigate
                  ? 'Previous message (↑)'
                  : 'No stream sequence — switch the live subscription mode to JetStream in Settings'}>
                  <button
                    type="button"
                    onClick={navigation.goPrev}
                    disabled={navigation.prevDisabled}
                    aria-label="Previous message"
                    data-testid="msg-nav-prev"
                    className="px-2 py-1 text-content-muted hover:text-gray-700 hover:bg-surface-tertiary transition-colors disabled:opacity-40 disabled:cursor-not-allowed disabled:hover:bg-transparent disabled:hover:text-content-muted"
                  >
                    {navigation.prevLoading ? (
                      <RefreshIcon className="w-4 h-4 animate-spin" />
                    ) : (
                      <ChevronUpIcon className="w-4 h-4" />
                    )}
                  </button>
                </Tooltip>
                <Tooltip content={navigation.canNavigate
                  ? 'Next message (↓)'
                  : 'No stream sequence — switch the live subscription mode to JetStream in Settings'}>
                  <button
                    type="button"
                    onClick={navigation.goNext}
                    disabled={navigation.nextDisabled}
                    aria-label="Next message"
                    data-testid="msg-nav-next"
                    className="px-2 py-1 border-l border-border text-content-muted hover:text-gray-700 hover:bg-surface-tertiary transition-colors disabled:opacity-40 disabled:cursor-not-allowed disabled:hover:bg-transparent disabled:hover:text-content-muted"
                  >
                    {navigation.nextLoading ? (
                      <RefreshIcon className="w-4 h-4 animate-spin" />
                    ) : (
                      <ChevronDownIcon className="w-4 h-4" />
                    )}
                  </button>
                </Tooltip>
              </span>
            )}
            {showActionSeparator && <span className="w-px h-[18px] bg-surface-hover mx-2" aria-hidden="true" />}
            {streamName && displayMessage?.sequence && (
              <BookmarkButton
                connectionId={connectionId!}
                streamName={streamName}
                sequence={displayMessage.sequence}
                subject={displayMessage.subject}
                dataPreview={jsonData ? JSON.stringify(jsonData).slice(0, 100) : displayMessage.data_base64?.slice(0, 100)}
              />
            )}
            {onResend && displayMessage?.subject && (
              <Tooltip content="Edit & resend: load this message into the publish form">
                <button
                  type="button"
                  onClick={() => void handleResend()}
                  aria-label="Resend message"
                  data-testid="resend-message"
                  className="p-1.5 rounded-md text-content-muted hover:text-accent hover:bg-accent-light transition-colors"
                >
                  <RefreshIcon className="w-4 h-4" />
                </button>
              </Tooltip>
            )}
            {streamName && displayMessage?.sequence && !displayMessage?.isLive && (
              <Tooltip content="Delete this message from the stream">
                <button
                  type="button"
                  onClick={requestDelete}
                  disabled={deleting}
                  aria-label="Delete message"
                  data-testid="delete-message"
                  className="p-1.5 rounded-md text-content-muted hover:text-status-error-text hover:bg-status-error-bg transition-colors disabled:opacity-50"
                >
                  <TrashIcon className="w-4 h-4" />
                </button>
              </Tooltip>
            )}
          </div>
        </div>
        <div className="text-sm text-content-secondary flex items-center">
          <button
            type="button"
            className="font-mono cursor-pointer hover:bg-surface-hover rounded px-1 -mx-1 transition-colors truncate min-w-0"
            onClick={() => {
              if (displayMessage?.subject) {
                void copyText(displayMessage.subject)
              }
            }}
            title="Click to copy subject"
          >
            {displayMessage?.subject || '—'}
          </button>
          <span className="mx-2">•</span>
          <span className="whitespace-nowrap">{displayMessage?.data_size || 0} bytes</span>
          <span className="mx-2">•</span>
          <span className="whitespace-nowrap">
            {displayMessage?.timestamp ? new Date(displayMessage.timestamp).toLocaleString() : '—'}
          </span>
          {isTruncated && (
            <>
              <span className="mx-2">•</span>
              <span
                className="px-2 py-0.5 text-xs font-medium bg-status-warning-light text-amber-800 rounded-full"
                title={`Server truncated this preview to keep the list response small. Original size: ${displayMessage?.data_size ?? 0} bytes.`}
              >
                Preview
              </span>
              <button
                type="button"
                disabled={loadingFull}
                onClick={handleLoadFull}
                className="ml-2 text-xs font-medium text-accent hover:text-accent-text disabled:opacity-50"
              >
                {loadingFull ? 'Loading…' : 'Load full payload'}
              </button>
              {loadFullError && (
                <span className="ml-2 text-xs text-status-error-text" title={loadFullError}>
                  failed
                </span>
              )}
            </>
          )}
        </div>
      </div>

      {/* Proto Type Info - hide for JSON messages without proto type */}
      {message?.content_type !== 'json' || selectedProtoType || selectedMessage?.decodedType ? (
        <div className="px-4 py-3 bg-surface-secondary border-b">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <span className="text-xs font-medium text-content-tertiary">Proto Type:</span>
              {selectedProtoType || selectedMessage?.decodedType ? (
                <span className="text-sm font-mono text-accent bg-accent-light px-2 py-1 rounded">
                  {selectedProtoType || selectedMessage?.decodedType}
                </span>
              ) : (
                <>
                  <span className="text-sm text-content-muted">Not configured</span>
                  {onOpenMappings && displayMessage?.subject && (
                    <button
                      onClick={() => onOpenMappings(getSubjectPattern(displayMessage.subject))}
                      className="ml-3 px-3 py-1.5 text-xs font-medium text-content-inverse bg-accent hover:bg-accent-hover rounded-md transition-colors flex items-center gap-1.5"
                    >
                      <PlusIcon className="w-3.5 h-3.5" />
                      Add Mapping
                    </button>
                  )}
                </>
              )}
            </div>
          </div>

          {(decodeError ||
            (selectedMessage?.decodeError && !decodedData && !message?.decoded && !willAutoDecode && !decoding)) && (
            <ErrorAlert message={decodeError || selectedMessage?.decodeError || ''} className="mt-2" />
          )}
        </div>
      ) : null}

      {/* Content — truncate notice renders INSIDE PayloadViewer so it sits next to the partial data it describes. */}
      <div className="flex-1 overflow-hidden">
        {message ? (
          <PayloadViewer
            rawData={message.data_base64}
            decodedData={decodedData}
            jsonData={jsonData}
            headers={message.headers}
            defaultMode={decodedData ? 'decoded' : decoding ? 'decoded' : message.content_type === 'json' ? 'json' : 'raw'}
            isDecoding={decoding}
            jsonIndentSize={display.jsonIndentSize}
            truncated={isTruncated}
            truncatedFullSize={displayMessage?.data_size}
            onLoadFull={handleLoadFull}
            loadingFull={loadingFull}
            loadFullError={loadFullError}
          />
        ) : null}
      </div>

      {displayMessage?.sequence != null && (
        <MessageDeleteDialog
          isOpen={deleteOpen}
          sequence={displayMessage.sequence}
          defaultSecure={behavior.secureDeleteDefault}
          isPending={deleting}
          onCancel={() => setDeleteOpen(false)}
          onConfirm={({ secure, dontAskAgain }) => {
            if (dontAskAgain) deleteConfirmation.disable()
            void performDelete(secure)
          }}
        />
      )}
    </div>
  )
}

