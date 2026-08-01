import { useState, useCallback, useEffect, useMemo, useRef } from 'react'
import { NavLink, Outlet, useParams, useOutletContext, useSearchParams, useLocation, useNavigate } from 'react-router-dom'
import { useStreamDetail } from '@/contexts/streams'
import { safeGetItem, safeSetItem } from '@/utils/safeStorage'
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
import { UsersIcon } from '@/components/ui'
import type { SelectedMessage } from '../messages/UnifiedMessageList'
import { useMessageNavigation } from '../messages/unified/useMessageNavigation'
import UnifiedMessageViewer from '../messages/UnifiedMessageViewer'
import type { ConnectionOutletContext } from '../common/ConnectedLayout'
import PublishHistory from './PublishHistory'
import StreamNotFoundState from './StreamNotFoundState'
import { isStreamNotFound } from './streamErrors'
import { subjectMatchesStream } from './publish/subjectPatternUtils'

const RIGHT_PANEL_WIDTH_KEY = 'nats_right_panel_width'
const EMPTY_HEADERS: HeaderDraft[] = []
const DEFAULT_RIGHT_PANEL_PCT = 50
const MIN_PANEL_PCT = 20
const MAX_PANEL_PCT = 80
const RESIZE_STEP_PCT = 2

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

  // Resizable right panel
  const [rightPanelPct, setRightPanelPct] = useState(() => {
    const saved = safeGetItem(RIGHT_PANEL_WIDTH_KEY)
    return saved ? Number(saved) : DEFAULT_RIGHT_PANEL_PCT
  })
  const isDragging = useRef(false)
  const containerRef = useRef<HTMLDivElement>(null)
  const latestPctRef = useRef(rightPanelPct)

  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      if (!isDragging.current || !containerRef.current) return
      const rect = containerRef.current.getBoundingClientRect()
      const pct = ((rect.right - e.clientX) / rect.width) * 100
      const clamped = Math.min(Math.max(pct, MIN_PANEL_PCT), MAX_PANEL_PCT)
      latestPctRef.current = clamped
      setRightPanelPct(clamped)
    }
    const handleMouseUp = () => {
      if (isDragging.current) {
        isDragging.current = false
        document.body.style.cursor = ''
        document.body.style.userSelect = ''
        safeSetItem(RIGHT_PANEL_WIDTH_KEY, String(Math.round(latestPctRef.current)))
      }
    }
    document.addEventListener('mousemove', handleMouseMove)
    document.addEventListener('mouseup', handleMouseUp)
    return () => {
      document.removeEventListener('mousemove', handleMouseMove)
      document.removeEventListener('mouseup', handleMouseUp)
    }
  }, [])

  const handleDragStart = useCallback((e: React.MouseEvent) => {
    e.preventDefault()
    isDragging.current = true
    document.body.style.cursor = 'col-resize'
    document.body.style.userSelect = 'none'
  }, [])

  const handleResizeKeyDown = useCallback((e: React.KeyboardEvent) => {
    const step = e.key === 'ArrowLeft' ? RESIZE_STEP_PCT : e.key === 'ArrowRight' ? -RESIZE_STEP_PCT : 0
    if (step === 0) return
    e.preventDefault()
    setRightPanelPct((prev) => {
      const next = Math.min(Math.max(prev + step, MIN_PANEL_PCT), MAX_PANEL_PCT)
      latestPctRef.current = next
      safeSetItem(RIGHT_PANEL_WIDTH_KEY, String(Math.round(next)))
      return next
    })
  }, [])

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

  // Config and Consumers tabs use full width (no right panel)
  const isFullWidthTab = isConfigTab || isConsumersTab

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
            <h2 className="text-sm font-semibold text-content-primary mb-3">
              Stream: {decodeURIComponent(streamName)}
            </h2>
            <div className="flex gap-1">
              <NavLink
                to={`${baseUrl}/messages`}
                end
                className={({ isActive }) =>
                  `px-4 py-2 text-sm font-medium rounded-t transition-colors flex items-center gap-1.5 ${
                    isActive
                      ? 'bg-surface-primary text-accent border-t border-x border-border'
                      : 'text-content-secondary hover:text-content-primary hover:bg-surface-tertiary'
                  }`
                }
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
                </svg>
                Messages
              </NavLink>
              <NavLink
                to={`${baseUrl}/config`}
                className={({ isActive }) =>
                  `px-4 py-2 text-sm font-medium rounded-t transition-colors flex items-center gap-1.5 ${
                    isActive
                      ? 'bg-surface-primary text-accent border-t border-x border-border'
                      : 'text-content-secondary hover:text-content-primary hover:bg-surface-tertiary'
                  }`
                }
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                </svg>
                Config
              </NavLink>
              <NavLink
                to={`${baseUrl}/consumers`}
                className={({ isActive }) =>
                  `px-4 py-2 text-sm font-medium rounded-t transition-colors flex items-center gap-1.5 ${
                    isActive
                      ? 'bg-surface-primary text-accent border-t border-x border-border'
                      : 'text-content-secondary hover:text-content-primary hover:bg-surface-tertiary'
                  }`
                }
              >
                <UsersIcon className="w-4 h-4" />
                Consumers
              </NavLink>
              <NavLink
                to={`${baseUrl}/publish`}
                className={({ isActive }) =>
                  `px-4 py-2 text-sm font-medium rounded-t transition-colors flex items-center gap-1.5 ${
                    isActive
                      ? 'bg-surface-primary text-accent border-t border-x border-border'
                      : 'text-content-secondary hover:text-content-primary hover:bg-surface-tertiary'
                  }`
                }
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" />
                </svg>
                Publish
              </NavLink>
            </div>
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
        } satisfies StreamViewOutletContext} />
      </main>

      {/* Right Panel - only for Messages and Publish tabs */}
      {!isFullWidthTab && (
        <>
          {/* Resize handle */}
          <div
            role="separator"
            aria-orientation="vertical"
            aria-label="Resize details panel"
            aria-valuenow={Math.round(rightPanelPct)}
            aria-valuemin={MIN_PANEL_PCT}
            aria-valuemax={MAX_PANEL_PCT}
            tabIndex={0}
            className="flex-shrink-0 cursor-col-resize group flex items-stretch focus:outline-none focus-visible:ring-2 focus-visible:ring-border-focus"
            onMouseDown={handleDragStart}
            onKeyDown={handleResizeKeyDown}
            style={{ padding: '0 2px' }}
          >
            <div className="w-px bg-surface-hover group-hover:bg-blue-400 group-active:bg-blue-500 group-focus-visible:bg-blue-500 transition-colors" />
          </div>
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
