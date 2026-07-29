import { useState, useEffect, useMemo, useRef, useCallback } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { CONNECTION_QUERY_PREFIX } from '@/hooks/useConnectionQuery'
import { useMessages, Subject } from '@/contexts/messages'
import { useDebouncedValue } from '@/hooks/useDebouncedValue'
import { type StreamScope, isScopeReady } from '@/stores/streamTabState'
import { setNavQuery } from '@/stores/streamTabState/messagesViewStore'
import { useStreamDetail } from '@/contexts/streams'
import type { Message } from '@/types/nats'
import type { SelectedMessage } from '@/types/messages'
import { getErrorMessage, getErrorReason } from '@/api/errors'
import ErrorAlert from '@/components/ui/ErrorAlert'
import { RefreshIcon } from '@/components/ui'
import {
  useDisplayPreferences,
  useMessagesPolicy,
  useLivePolicy,
  useUpdateSettings,
  useSettings,
} from '@/contexts/settings'
import MessageDiffViewer from './MessageDiffViewer'
import ExportDialog from './ExportDialog'
import { parseStartDate } from './jumpToTime'
import StreamStatsHeader from './StreamStatsHeader'
import type { FilterValues } from './AdvancedFilters'
import { MessageToolbar } from './unified/MessageToolbar'
import { CompareModeBar } from './unified/CompareModeBar'
import {
  NoStreamSelected,
  MessagesLoading,
  RealtimeStatusBar,
  EmptyMessagesState,
  WorkQueueWarning,
  WorkQueueRealtimeNotice,
} from './unified/MessageListStates'
import { MessageVirtualTable } from './unified/MessageVirtualTable'
import { useLiveSubscription } from './unified/useLiveSubscription'
import { useLoadMoreMessages } from './unified/useLoadMoreMessages'
import { toSelectedHistoryMessage } from './unified/selectedMessage'
import {
  liveToMessage,
  isExportableMessage,
  type LiveMessage,
  type LiveMessageLimit,
  type ViewMode,
} from './unified/messageListUtils'

interface UnifiedMessageListProps {
  streamName: string | null
  connectionId: string | null
  onSelectMessage: (message: SelectedMessage | null) => void
  selectedMessageId?: string | null
  /** Stream-scoped store key; enables navQuery publishing when provided. */
  scope?: StreamScope
}

// Re-export: SelectedMessage lives in @/types/messages (so stores avoid the
// stores->ui boundary); keeps existing imports working.
export type { SelectedMessage } from '@/types/messages'

const emptyFilters: FilterValues = {
  subject: '',
  startSequence: null,
  startDate: null,
  contentFilter: '',
}

export default function UnifiedMessageList({
  streamName,
  connectionId,
  onSelectMessage,
  selectedMessageId,
  scope,
}: UnifiedMessageListProps) {
  const queryClient = useQueryClient()
  const display = useDisplayPreferences()
  const msgSettings = useMessagesPolicy()
  const liveSettings = useLivePolicy()
  const updateSettingsMutation = useUpdateSettings()
  const { isSuccess: settingsLoaded } = useSettings()

  const isCompact = display.density === 'compact'
  const rowHeight = isCompact ? 36 : 52
  const cellPadding = isCompact ? 'px-3 py-1' : 'px-3 py-2'

  const autoScrollRef = useRef(display.autoScrollLive)
  useEffect(() => {
    autoScrollRef.current = display.autoScrollLive
  }, [display.autoScrollLive])

  const [mode, setMode] = useState<ViewMode>('history')
  const [limit, setLimit] = useState(50)
  const settingsAppliedRef = useRef(false)
  useEffect(() => {
    if (settingsAppliedRef.current || !settingsLoaded) return
    settingsAppliedRef.current = true
    if (display.defaultViewMode) setMode(display.defaultViewMode as ViewMode)
    if (msgSettings.defaultPageSize) setLimit(msgSettings.defaultPageSize)
  }, [settingsLoaded, display.defaultViewMode, msgSettings.defaultPageSize])

  const [filters, setFilters] = useState<FilterValues>(emptyFilters)
  const [showFiltersPanel, setShowFiltersPanel] = useState(false)

  const [showExportDialog, setShowExportDialog] = useState(false)
  const [compareMode, setCompareMode] = useState(false)
  const [compareMessages, setCompareMessages] = useState<[Message | null, Message | null]>([null, null])
  const [showDiffViewer, setShowDiffViewer] = useState(false)

  const debouncedSubjectFilter = useDebouncedValue(filters.subject, 300)
  const debouncedContentFilter = useDebouncedValue(filters.contentFilter, 300)

  // Jump-to-time: a startDate switches paging from sequence to a time anchor
  // (mutually exclusive server-side; reads forward).
  const jumpStartMs = parseStartDate(filters.startDate)
  const jumpActive = jumpStartMs != null

  const effectiveDirection = jumpActive
    ? ('forward' as const)
    : (msgSettings.defaultDirection as 'backward' | 'forward')

  // Publish the effective query context for arrow navigation (StreamView) —
  // it must fetch with exactly the filters/direction the list shows.
  useEffect(() => {
    if (!scope || !isScopeReady(scope)) return
    setNavQuery(scope, {
      subjectFilter: debouncedSubjectFilter || undefined,
      contentFilter: debouncedContentFilter || undefined,
      direction: effectiveDirection,
    })
  }, [scope, debouncedSubjectFilter, debouncedContentFilter, effectiveDirection])

  const { data: historyData, isLoading, error, refetch, isFetching } = useMessages(streamName, {
    connection_id: connectionId,
    limit,
    subject_filter: debouncedSubjectFilter || undefined,
    content_filter: debouncedContentFilter || undefined,
    start_seq: jumpActive ? undefined : (filters.startSequence ?? undefined),
    start_time: jumpActive ? jumpStartMs : undefined,
    direction: effectiveDirection,
  }, { enabled: mode === 'history' })

  // Follow-up pages chained off the base query; reset whenever it changes.
  const {
    messages: historyMessages,
    hasMore,
    loadMore,
    isLoadingMore,
  } = useLoadMoreMessages({
    streamName,
    connectionId,
    baseData: historyData,
    limit,
    subjectFilter: debouncedSubjectFilter || undefined,
    contentFilter: debouncedContentFilter || undefined,
    direction: effectiveDirection,
  })

  const { data: streamDetail } = useStreamDetail(streamName, connectionId)
  const availableSubjects = useMemo(
    () => (streamDetail?.subjects ? [...streamDetail.subjects].sort() : []),
    [streamDetail?.subjects],
  )

  // WorkQueue is safe for live view: server downgrades the consumer to Core
  // NATS pub/sub (no ack/consume); banner explains no backlog replay.
  const isWorkQueueStream = streamDetail?.config?.retention?.toLowerCase() === 'workqueue'

  // Live subscription (only runs when mode === 'realtime')
  const {
    liveMessages,
    liveLimit,
    setLiveLimit,
    wsStatus,
    wsError,
    isPaused,
    togglePause,
    newMessageIds,
    clearMessages: clearLive,
  } = useLiveSubscription({
    connectionId,
    streamName,
    enabled: mode === 'realtime',
    initialLimit: 50,
    maxDisplayRate: liveSettings.maxDisplayRate,
    subjectFilter: filters.subject || undefined,
  })

  // Clear selection only on real stream change, not on mount — mount-time
  // clearing would defeat the per-stream view store's restore.
  const prevStreamRef = useRef<string | null | undefined>(undefined)
  useEffect(() => {
    if (prevStreamRef.current !== undefined && prevStreamRef.current !== streamName) {
      onSelectMessage(null)
    }
    prevStreamRef.current = streamName
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [streamName])

  const filteredLiveMessages = useMemo(() => {
    if (!filters.subject) return liveMessages
    const pattern = filters.subject
    if (pattern.includes('*') || pattern.includes('>')) {
      return liveMessages.filter((msg) => Subject.fromTrusted(msg.subject).matchesPattern(pattern))
    }
    const lower = pattern.toLowerCase()
    return liveMessages.filter((msg) => msg.subject.toLowerCase().includes(lower))
  }, [liveMessages, filters.subject])

  const historyLoading = mode === 'history' && (isLoading || (isFetching && !historyData))
  const displayMessages: Array<Message | LiveMessage> =
    mode === 'history' ? historyMessages : filteredLiveMessages

  const exportMessages: Message[] = useMemo(
    () =>
      (mode === 'history' ? historyMessages : filteredLiveMessages.map(liveToMessage)).filter(
        isExportableMessage,
      ),
    [mode, historyMessages, filteredLiveMessages],
  )

  const handleFilterChange = (key: keyof FilterValues) => {
    setFilters((prev) => ({
      ...prev,
      [key]: key === 'startSequence' ? null : key === 'startDate' ? null : '',
    }))
  }

  const handleClearAllFilters = () => setFilters(emptyFilters)

  const handleClearLive = () => {
    clearLive()
    onSelectMessage(null)
  }

  // useCallback for stable identities — otherwise React.memo on rows falls
  // through and they re-render on every filter keystroke.
  const isCompareSelected = useCallback(
    (msg: Message) =>
      compareMessages[0]?.sequence === msg.sequence ||
      compareMessages[1]?.sequence === msg.sequence,
    [compareMessages],
  )

  const handleCompareSelect = useCallback((msg: Message) => {
    if (compareMessages[0]?.sequence === msg.sequence) {
      setCompareMessages([compareMessages[1], null])
      return
    }
    if (compareMessages[1]?.sequence === msg.sequence) {
      setCompareMessages([compareMessages[0], null])
      return
    }
    if (!compareMessages[0]) setCompareMessages([msg, null])
    else setCompareMessages([compareMessages[0], msg])
  }, [compareMessages])

  const handleSelectHistoryMessage = useCallback((msg: Message) => {
    onSelectMessage(toSelectedHistoryMessage(msg))
  }, [onSelectMessage])

  const handleSelectLiveMessage = useCallback((msg: LiveMessage) => {
    onSelectMessage({
      id: msg.id,
      sequence: msg.sequence,
      subject: msg.subject,
      timestamp: msg.timestamp,
      data_base64: msg.data_base64,
      data_size: msg.data_size,
      content_type: msg.content_type,
      headers: msg.headers,
      decoded: msg.decoded,
      decodedType: msg.decodedType,
      decodeError: msg.decodeError,
      truncated: (msg as Message).truncated ?? undefined,
      isLive: true,
    })
  }, [onSelectMessage])

  const handleToggleCompare = () => {
    setCompareMode((prev) => {
      if (prev) {
        setCompareMessages([null, null])
        setShowDiffViewer(false)
      }
      return !prev
    })
  }

  const handleRefetch = () => {
    refetch()
    queryClient.invalidateQueries({
      queryKey: [CONNECTION_QUERY_PREFIX, connectionId, 'stream', streamName],
    })
    queryClient.invalidateQueries({
      queryKey: [CONNECTION_QUERY_PREFIX, connectionId, 'streamStats', streamName],
    })
  }

  const handleMaxDisplayRateChange = (rate: number) => {
    updateSettingsMutation.mutate({ live: { maxDisplayRate: rate } })
  }

  if (!streamName || !connectionId) {
    return <NoStreamSelected />
  }

  const isWorkQueue = streamDetail?.config?.retention?.toLowerCase() === 'workqueue' && mode === 'history'
  // The backend refuses on purpose (would drain the queue); WorkQueueWarning
  // below already explains it and offers the fix, so the red alert is noise.
  const isWorkQueueConsumerError = isWorkQueue && getErrorReason(error) === 'NATS_WORKQUEUE_CONSUMER_NOT_ALLOWED'

  return (
    <div className="flex flex-col flex-1 min-h-0">
      <StreamStatsHeader streamName={streamName} connectionId={connectionId} />

      <MessageToolbar
        mode={mode}
        onModeChange={setMode}
        filters={filters}
        onFiltersChange={setFilters}
        onRemoveFilter={handleFilterChange}
        onClearAllFilters={handleClearAllFilters}
        showFiltersPanel={showFiltersPanel}
        onToggleFiltersPanel={() => setShowFiltersPanel((v) => !v)}
        availableSubjects={availableSubjects}
        messageCount={displayMessages.length}
        limit={limit}
        onLimitChange={setLimit}
        onRefetch={handleRefetch}
        isFetching={isFetching}
        liveLimit={liveLimit}
        onLiveLimitChange={(v) => setLiveLimit(v as LiveMessageLimit)}
        wsStatus={wsStatus}
        isPaused={isPaused}
        onTogglePause={togglePause}
        onClearLive={handleClearLive}
        maxDisplayRate={liveSettings.maxDisplayRate}
        onMaxDisplayRateChange={handleMaxDisplayRateChange}
        compareMode={compareMode}
        onToggleCompareMode={handleToggleCompare}
        onOpenExport={() => setShowExportDialog(true)}
      />

      {compareMode && (
        <CompareModeBar
          messageA={compareMessages[0]}
          messageB={compareMessages[1]}
          onOpenDiff={() => setShowDiffViewer(true)}
          onCancel={() => {
            setCompareMode(false)
            setCompareMessages([null, null])
          }}
        />
      )}

      {mode === 'history' && error && !isWorkQueueConsumerError && (
        <div className="px-4 py-2">
          <ErrorAlert message={getErrorMessage(error)} />
        </div>
      )}
      {mode === 'realtime' && wsError && (
        <div className="px-4 py-2">
          <ErrorAlert message={wsError} />
        </div>
      )}

      {isWorkQueue && <WorkQueueWarning onSwitchToRealtime={() => setMode('realtime')} />}

      {mode === 'realtime' && isWorkQueueStream && <WorkQueueRealtimeNotice />}

      {historyLoading && <MessagesLoading />}

      {mode === 'history' && jumpActive && !historyLoading && (
        <div
          data-testid="jump-resolved"
          className="px-4 py-2 bg-accent-light border-b border-blue-100 text-xs text-blue-800 flex items-center gap-1.5"
        >
          <svg className="w-3.5 h-3.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          {displayMessages.length > 0 ? (
            <span>
              Jumped to {new Date(jumpStartMs!).toLocaleString()} → first message{' '}
              <span className="font-mono font-semibold" data-testid="jump-resolved-seq">
                #{(displayMessages[0] as Message).sequence}
              </span>
            </span>
          ) : (
            <span>No messages at or after {new Date(jumpStartMs!).toLocaleString()}</span>
          )}
        </div>
      )}

      {mode === 'realtime' && wsStatus !== 'connected' && (
        <RealtimeStatusBar status={wsStatus} />
      )}

      {!historyLoading &&
        (displayMessages.length === 0 ? (
          <EmptyMessagesState subjectFilter={filters.subject} isRealtime={mode === 'realtime'} />
        ) : (
          <>
            <MessageVirtualTable
              messages={displayMessages}
              mode={mode}
              selectedMessageId={selectedMessageId}
              newMessageIds={newMessageIds}
              compareMode={compareMode}
              isCompareSelected={isCompareSelected}
              rowHeight={rowHeight}
              cellPadding={cellPadding}
              streamName={streamName}
              connectionId={connectionId}
              timestampFormat={display.timestampFormat as 'relative' | 'absolute' | 'iso'}
              autoScrollRef={autoScrollRef}
              onSelectHistory={handleSelectHistoryMessage}
              onSelectLive={handleSelectLiveMessage}
              onCompareSelect={handleCompareSelect}
            />
            {mode === 'history' && hasMore && (
              <div className="p-2 bg-surface-secondary border-t flex items-center justify-center gap-3 text-sm text-content-secondary">
                <span>Showing {displayMessages.length} messages.</span>
                <button
                  onClick={loadMore}
                  disabled={isLoadingMore}
                  data-testid="load-more"
                  className="px-3 py-1 text-xs font-medium border border-border-strong text-gray-700 rounded-md hover:bg-surface-tertiary disabled:opacity-50 disabled:cursor-not-allowed transition-colors inline-flex items-center gap-1.5"
                >
                  {isLoadingMore && <RefreshIcon className="w-3 h-3 animate-spin" />}
                  {isLoadingMore ? 'Loading…' : 'Load more'}
                </button>
              </div>
            )}
          </>
        ))}

      {showExportDialog && (
        <ExportDialog
          isOpen={showExportDialog}
          onClose={() => setShowExportDialog(false)}
          messages={exportMessages}
          streamName={streamName || ''}
          connectionId={connectionId}
          streamFirstSeq={streamDetail?.state?.first_seq}
          totalCount={streamDetail?.state?.messages}
        />
      )}

      {showDiffViewer && (
        <MessageDiffViewer
          messageA={compareMessages[0]}
          messageB={compareMessages[1]}
          onClose={() => setShowDiffViewer(false)}
        />
      )}
    </div>
  )
}
