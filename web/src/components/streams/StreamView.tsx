import { useCallback, useEffect, useMemo } from 'react'
import { Outlet, useParams, useOutletContext, useSearchParams, useLocation, useNavigate } from 'react-router-dom'
import { useStreamDetail } from '@/contexts/streams'
import { useResizablePanel } from '@/hooks/useResizablePanel'
import {
  type StreamScope,
  isScopeReady,
} from '@/stores/streamTabState'
import {
  usePublishDraftEntry,
  getPublishDraftEntry,
  getPatternDraft,
  setLastPattern,
  setPatternDraft,
  clearPatternDraft,
  type HeaderDraft,
} from '@/stores/streamTabState/publishDraftStore'
import { useMessagesViewEntry } from '@/stores/streamTabState/messagesViewStore'
import type { SelectedMessage } from '../messages/UnifiedMessageList'
import { useMessageNavigation } from '../messages/unified/useMessageNavigation'
import UnifiedMessageViewer from '../messages/UnifiedMessageViewer'
import type { ConnectionOutletContext } from '../common/ConnectedLayout'
import { ResizeHandle } from '../common/ResizeHandle'
import PublishHistory from './PublishHistory'
import StreamNotFoundState from './StreamNotFoundState'
import StreamTabs from './StreamTabs'
import { isStreamNotFound } from './streamErrors'
import { subjectMatchesStream } from './publish/subjectPatternUtils'
import type { StreamPublishFeatures } from './publish/publishOptions'

const EMPTY_HEADERS: HeaderDraft[] = []

export interface StreamViewOutletContext {
  /** Stream-scoped storage key (connection URL + stream name). */
  scope: StreamScope
  connectionId: string
  streamName: string
  selectedMessage: SelectedMessage | null
  setSelectedMessage: (msg: SelectedMessage | null) => void
  handleOpenMappings: (subjectPattern?: string) => void
  // Publish form state — sourced from publishDraftStore via StreamView.
  publishPattern: string
  setPublishPattern: (pattern: string) => void
  publishWildcards: string[]
  setPublishWildcards: (wildcards: string[]) => void
  publishMessageJson: string
  setPublishMessageJson: (json: string) => void
  publishHeaders: HeaderDraft[]
  setPublishHeaders: (headers: HeaderDraft[]) => void
  subjects: string[]
  /** Stream's max_msg_size limit in bytes (0/undefined = unlimited). */
  streamMaxMsgSize?: number
  /** Stream flags that gate JetStream publish options; undefined until the stream loads. */
  streamPublishFeatures?: StreamPublishFeatures
}

export default function StreamView() {
  const { streamName } = useParams<{ streamName: string }>()
  const { connectionId, currentConnection, handleOpenMappings } = useOutletContext<ConnectionOutletContext>()
  const [, setSearchParams] = useSearchParams()
  const navigate = useNavigate()

  // The single per-stream scope used by every store keyed by (connection,
  // stream).
  const scope: StreamScope = useMemo(
    () => ({
      connectionUrl: currentConnection?.urls[0] || null,
      streamName: streamName || '',
    }),
    [currentConnection?.urls, streamName],
  )

  // Selected message state — persisted per scope (sessionStorage).
  const [messagesView, setMessagesView] = useMessagesViewEntry(scope)
  const selectedMessage = messagesView.selectedMessage

  // Publish state via the scoped store; derive the active pattern's draft
  // below.
  const [publishEntry] = usePublishDraftEntry(scope)
  const publishPattern = publishEntry.lastPattern
  const activeDraft = useMemo(
    () => getPatternDraft(publishEntry, publishPattern),
    [publishEntry, publishPattern],
  )
  const publishWildcards = activeDraft.wildcards
  const publishMessageJson = activeDraft.messageJson
  const publishHeaders = activeDraft.headers ?? EMPTY_HEADERS

  const setPublishPattern = useCallback(
    (pattern: string) => {
      if (!isScopeReady(scope)) return
      setLastPattern(scope, pattern)
    },
    [scope],
  )

  // Setters read lastPattern fresh from the store (not closed-over state) so
  // same-tick setPattern+setJson calls land in the right pattern's draft.
  const setPublishWildcards = useCallback(
    (wildcards: string[]) => {
      if (!isScopeReady(scope)) return
      const pattern = getPublishDraftEntry(scope).lastPattern
      if (!pattern) return
      setPatternDraft(scope, pattern, { wildcards })
    },
    [scope],
  )

  const setPublishMessageJson = useCallback(
    (messageJson: string) => {
      if (!isScopeReady(scope)) return
      const pattern = getPublishDraftEntry(scope).lastPattern
      if (!pattern) return
      setPatternDraft(scope, pattern, { messageJson })
    },
    [scope],
  )

  const setPublishHeaders = useCallback(
    (headers: HeaderDraft[]) => {
      if (!isScopeReady(scope)) return
      const pattern = getPublishDraftEntry(scope).lastPattern
      if (!pattern) return
      setPatternDraft(scope, pattern, { headers })
    },
    [scope],
  )

  const { rightPanelPct, containerRef, separatorProps } = useResizablePanel()

  const { data: streamDetail, error: streamError } = useStreamDetail(streamName ?? null, connectionId)

  useEffect(() => {
    if (!isScopeReady(scope)) return
    const currentSubjects = streamDetail?.subjects
    if (currentSubjects && publishPattern && !subjectMatchesStream(publishPattern, currentSubjects)) {
      clearPatternDraft(scope, publishPattern)
      setLastPattern(scope, '')
    }
  }, [streamDetail?.subjects, publishPattern, scope])

  // Select a message: sync to URL (shareable) + persist for survival across
  // navigation.
  const handleSelectMessage = useCallback((msg: SelectedMessage | null) => {
    setMessagesView({ selectedMessage: msg })
    setSearchParams((prev) => {
      const newParams = new URLSearchParams(prev)
      if (msg) {
        newParams.set('msg', msg.id)
      } else {
        newParams.delete('msg')
      }
      return newParams
    }, { replace: true })
  }, [setSearchParams, setMessagesView])

  // Edit & resend: fill the publish draft from a message, then jump to the
  // Publish tab.
  const handleResend = useCallback(
    (draft: { pattern: string; wildcards: string[]; messageJson: string; headers: HeaderDraft[] }) => {
      if (!streamName) return
      setPublishPattern(draft.pattern)
      setPublishWildcards(draft.wildcards)
      setPublishMessageJson(draft.messageJson)
      setPublishHeaders(draft.headers)
      navigate(`/streams/${encodeURIComponent(streamName)}/publish`)
    },
    [streamName, navigate, setPublishPattern, setPublishWildcards, setPublishMessageJson, setPublishHeaders],
  )

  // Determine current tab from URL
  const location = useLocation()
  const isPublishTab = location.pathname.endsWith('/publish')
  const isConfigTab = location.pathname.endsWith('/config')
  const isConsumersTab = location.pathname.endsWith('/consumers')
  const isRelationsTab = location.pathname.endsWith('/relations')

  // Config, Consumers and Relations tabs use full width (no right panel)
  const isFullWidthTab = isConfigTab || isConsumersTab || isRelationsTab

  const streamMissing = isStreamNotFound(streamError)

  // Arrow navigation — must be called before the early return (hooks rule).
  const isMessagesTab = !isPublishTab && !isFullWidthTab
  const navigation = useMessageNavigation({
    streamName: streamName ?? null,
    connectionId: connectionId ?? null,
    selectedMessage,
    navQuery: messagesView.navQuery,
    onSelectMessage: handleSelectMessage,
    keyboardEnabled: isMessagesTab && !!selectedMessage,
  })

  if (!connectionId || !streamName) {
    return null
  }

  if (streamMissing) {
    return (
      <main className="flex-1 flex items-center justify-center bg-surface-primary" id="main-content" role="main">
        <StreamNotFoundState streamName={streamName} />
      </main>
    )
  }

  const baseUrl = `/streams/${encodeURIComponent(streamName)}`

  return (
    <div ref={containerRef} className="flex flex-1 overflow-hidden">
      {/* Main Content */}
      <main
        className="bg-surface-primary flex flex-col overflow-hidden"
        style={isFullWidthTab ? { flex: 1 } : { width: `${100 - rightPanelPct}%` }}
        id="main-content"
        role="main"
        aria-label="Stream content"
      >
        {/* Header with Tabs */}
        <div className="border-b bg-surface-secondary">
          <div className="px-4 py-3">
            <h2 className="text-sm font-semibold text-content-primary mb-3 truncate" title={decodeURIComponent(streamName)}>
              Stream: {decodeURIComponent(streamName)}
            </h2>
            <StreamTabs baseUrl={baseUrl} />
          </div>
        </div>

        {/* Tab Content */}
        <Outlet context={{
          scope,
          connectionId,
          streamName,
          selectedMessage,
          setSelectedMessage: handleSelectMessage,
          handleOpenMappings,
          publishPattern,
          setPublishPattern,
          publishWildcards,
          setPublishWildcards,
          publishMessageJson,
          setPublishMessageJson,
          publishHeaders,
          setPublishHeaders,
          subjects: streamDetail?.subjects || [],
          streamMaxMsgSize: streamDetail?.config?.max_msg_size,
          streamPublishFeatures: streamDetail?.config
            ? {
                msgTtl: !!streamDetail.config.allow_msg_ttl,
                msgSchedules: !!streamDetail.config.allow_msg_schedules,
                msgCounter: !!streamDetail.config.allow_msg_counter,
              }
            : undefined,
        } satisfies StreamViewOutletContext} />
      </main>

      {/* Right Panel - only for Messages and Publish tabs */}
      {!isFullWidthTab && (
        <>
          {/* Resize handle */}
          <ResizeHandle {...separatorProps} />
          <aside
            className="bg-surface-secondary flex flex-col overflow-hidden"
            style={{ width: `${rightPanelPct}%` }}
            role="complementary"
            aria-label="Details panel"
          >
            {isPublishTab ? (
              <PublishHistory
                streamName={streamName}
                connectionId={connectionId}
                connectionUrl={currentConnection?.urls[0] || null}
                subjects={streamDetail?.subjects || []}
              />
            ) : (
              <UnifiedMessageViewer
                streamName={streamName}
                connectionId={connectionId}
                selectedMessage={selectedMessage}
                onOpenMappings={handleOpenMappings}
                onDeleted={() => handleSelectMessage(null)}
                onResend={handleResend}
                navigation={navigation}
              />
            )}
          </aside>
        </>
      )}
    </div>
  )
}
