import { useCallback, useMemo, useRef, useState } from 'react'
import { useOutletContext } from 'react-router-dom'
import { useMappingItems } from '@/contexts/mappings'
import { useDisplayPreferences } from '@/contexts/settings'
import { useResizablePanel } from '@/hooks/useResizablePanel'
import { useSubscribeDraft } from '@/stores/subscribeDraftStore'
import type { SelectedMessage } from '@/types/messages'
import { EmptyState, LockClosedIcon, SignalIcon } from '@/components/ui'
import ErrorAlert from '@/components/ui/ErrorAlert'
import { AccessDeniedState } from '../common/access/AccessDeniedState'
import type { ConnectionOutletContext } from '../common/ConnectedLayout'
import { ResizeHandle } from '../common/ResizeHandle'
import UnifiedMessageViewer from '../messages/UnifiedMessageViewer'
import type { ResendDraft } from '../messages/resend'
import { MessageVirtualTable } from '../messages/unified/MessageVirtualTable'
import type { LiveMessage, WsStatus } from '../messages/unified/messageListUtils'
import { buildSubject } from '../streams/publish/subjectPatternUtils'
import { CorePublishDialog, type CorePublishDraft } from './CorePublishDialog'
import { SubjectBar } from './SubjectBar'
import { useSubscribeSession } from './subscribeSession'
import { SubscribeToolbar } from './SubscribeToolbar'
import { filterReceived } from './subscribeUtils'

const NO_COMPARE = () => false
const NOOP = () => {}

interface PillProps {
  running: boolean
  hasFeed: boolean
  status: WsStatus
  paused: boolean
  denied: boolean
  failed: boolean
}

function pillState({ running, hasFeed, status, paused, denied, failed }: PillProps) {
  if (!running) return { label: hasFeed ? 'Stopped' : 'Not subscribed', dot: 'bg-content-muted', text: 'text-content-tertiary', pulse: false }
  if (denied) return { label: 'No permission', dot: null, text: 'text-content-secondary', pulse: false }
  if (failed) return { label: 'Disconnected', dot: 'bg-status-error-border', text: 'text-status-error-text', pulse: false }
  if (status !== 'connected') {
    const label = status === 'reconnecting' ? 'Reconnecting…' : 'Connecting…'
    return { label, dot: 'bg-status-warning-border', text: 'text-status-warning-text', pulse: true }
  }
  if (paused) return { label: 'Paused', dot: 'bg-status-warning-border', text: 'text-status-warning-text', pulse: false }
  return { label: 'Live', dot: 'bg-status-success-border', text: 'text-status-success-text', pulse: true }
}

function StatusPill(props: PillProps) {
  const state = pillState(props)
  return (
    <span
      role="status"
      className={`inline-flex items-center gap-1.5 rounded-full border border-border bg-surface-primary px-2.5 py-1 text-xs font-medium ${state.text}`}
      data-testid="subscribe-status"
    >
      {state.dot ? (
        <span aria-hidden="true" className={`w-2 h-2 rounded-full ${state.dot} ${state.pulse ? 'animate-pulse motion-reduce:animate-none' : ''}`} />
      ) : (
        <LockClosedIcon className="w-3 h-3" />
      )}
      {state.label}
    </span>
  )
}

function toSelected(msg: LiveMessage): SelectedMessage {
  return {
    id: msg.id,
    subject: msg.subject,
    timestamp: msg.timestamp,
    data_base64: msg.data_base64,
    data_size: msg.data_size,
    content_type: msg.content_type,
    headers: msg.headers,
    decoded: msg.decoded,
    decodedType: msg.decodedType,
    decodedAuto: msg.decodedAuto,
    decodedSourceId: msg.decodedSourceId,
    decodeError: msg.decodeError,
    truncated: msg.truncated,
    reply: msg.reply,
    isLive: true,
  }
}

export default function SubscribePage() {
  const { connectionId, handleOpenMappings } = useOutletContext<ConnectionOutletContext>()
  const [draft, updateDraft] = useSubscribeDraft(connectionId)
  const session = useSubscribeSession()
  const { running, allDenied, start, stop, live, query, setQuery, subjectFilter, setSubjectFilter, selected, setSelected } = session
  const [dialog, setDialog] = useState<{ mode: 'resend' | 'reply'; initial: CorePublishDraft } | null>(null)

  const display = useDisplayPreferences()
  const { data: mappings = [] } = useMappingItems()
  const { rightPanelPct, containerRef, separatorProps } = useResizablePanel()
  const autoScrollRef = useRef(true)

  const suggestions = useMemo(
    () => [...new Set([...draft.recentSubjects, ...mappings.map((m) => m.pattern)])],
    [draft.recentSubjects, mappings],
  )

  const visible = useMemo(
    () => filterReceived(live.liveMessages, query, subjectFilter, session.muted),
    [live.liveMessages, query, subjectFilter, session.muted],
  )

  const received = useMemo(() => Object.values(live.subjectCounts).reduce((sum, n) => sum + n, 0), [live.subjectCounts])
  const filtering = query.trim() !== '' || subjectFilter !== null

  const onSubjectsChange = useCallback((subjects: string[]) => updateDraft({ subjects }), [updateDraft])

  const clear = () => {
    live.clearMessages()
    setSelected(null)
    setSubjectFilter(null)
  }

  const selectLive = useCallback((msg: LiveMessage) => setSelected(toSelected(msg)), [setSelected])
  const closeDetails = useCallback(() => setSelected(null), [setSelected])

  const openResend = useCallback((d: ResendDraft) => {
    setDialog({
      mode: 'resend',
      initial: { subject: buildSubject(d.pattern, d.wildcards), payload: d.verbatim ?? d.messageJson, headers: d.headers },
    })
  }, [])

  const openReply = useCallback((message: SelectedMessage) => {
    if (!message.reply) return
    setDialog({ mode: 'reply', initial: { subject: message.reply, payload: '', headers: [] } })
  }, [])

  const hasFeed = running || live.liveMessages.length > 0
  const isCompact = display.density === 'compact'

  const feed = () => {
    if (allDenied) {
      return (
        <AccessDeniedState
          check={{ status: 'denied', operation: 'subscribe', subject: live.deniedSubjects.join(', ') }}
          title="No permission to subscribe"
          description={`The server refused ${live.deniedSubjects.length === 1 ? 'the subject' : 'every subject'} in this subscription for your NATS user.`}
        />
      )
    }
    if (live.liveMessages.length === 0) {
      return (
        <div className="flex-1 flex items-center justify-center">
          <EmptyState
            icon={<SignalIcon className="w-full h-full" />}
            title="Waiting for messages…"
            description={`Listening on ${draft.subjects.join(', ')}. Core NATS keeps no history, so only messages published from now on show up.`}
          />
        </div>
      )
    }
    if (visible.length === 0) {
      return (
        <div className="flex-1 flex items-center justify-center">
          <EmptyState title="No received message matches" description="Clear the search, pick another subject or unmute one." />
        </div>
      )
    }
    return (
      <MessageVirtualTable
        messages={visible}
        mode="realtime"
        selectedMessageId={selected?.id}
        newMessageIds={live.newMessageIds}
        compareMode={false}
        isCompareSelected={NO_COMPARE}
        rowHeight={isCompact ? 36 : 44}
        cellPadding={isCompact ? 'px-3 py-1' : 'px-3 py-2'}
        streamName=""
        connectionId={connectionId}
        timestampFormat={display.timestampFormat as 'relative' | 'absolute' | 'iso'}
        showSequence={false}
        showPayload
        autoScrollRef={autoScrollRef}
        onSelectHistory={NOOP}
        onSelectLive={selectLive}
        onCompareSelect={NOOP}
      />
    )
  }

  return (
    <div className="flex-1 flex flex-col overflow-hidden bg-surface-primary">
      <div className={`px-6 border-b border-border ${hasFeed ? 'pt-4 pb-3' : 'pt-5 pb-4'}`}>
        <div className="flex items-start justify-between gap-4">
          <div>
            <h2 className="text-lg font-semibold text-content-primary">Subscribe</h2>
            {!hasFeed && (
              <p className="text-sm text-content-tertiary mt-0.5">
                Watch any subject over core NATS. Nothing is stored: the feed starts when you subscribe.
              </p>
            )}
          </div>
          <StatusPill
            running={running}
            hasFeed={hasFeed}
            status={live.wsStatus}
            paused={live.isPaused}
            denied={allDenied}
            failed={live.wsStatus === 'disconnected' && !!live.wsError}
          />
        </div>
        <SubjectBar
          subjects={draft.subjects}
          onSubjectsChange={onSubjectsChange}
          deniedSubjects={running ? live.deniedSubjects : []}
          suggestions={suggestions}
          recentSubjects={draft.recentSubjects}
          running={running}
          onStart={start}
          onStop={stop}
          showQuickAdd={!running}
        />
        {draft.subjects.includes('>') && (
          <p className="mt-2 text-xs text-content-tertiary">
            Natscope hides subjects starting with $ (such as $JS, $SYS, $KV) and _INBOX under &gt;. Add them by name to see them.
          </p>
        )}
      </div>

      {!hasFeed ? (
        <div className="flex-1 flex items-center justify-center">
          <EmptyState
            size="lg"
            icon={<SignalIcon className="w-full h-full" />}
            title="Subscribe to any subject"
            description="Add one or more subjects and press Start. orders.* matches one token, orders.> everything below it. Core NATS keeps no history, so you see messages from the moment you subscribe."
          />
        </div>
      ) : (
        <div ref={containerRef} className="flex flex-1 min-h-0 overflow-hidden">
          <section aria-label="Received messages" className="flex-1 min-w-0 flex flex-col">
            {!allDenied && (
              <SubscribeToolbar
                query={query}
                onQueryChange={setQuery}
                subjectCounts={live.subjectCounts}
                subjectFilter={subjectFilter}
                onSubjectFilterChange={setSubjectFilter}
                muted={session.muted}
                onMute={session.mute}
                onUnmute={session.unmute}
                counts={{
                  received,
                  shown: live.liveMessages.length,
                  matching: filtering ? visible.length : null,
                  skipped: live.messagesDropped,
                  msgPerSecond: live.msgPerSecond,
                }}
                running={running}
                isPaused={live.isPaused}
                pausedCount={live.pausedCount}
                onTogglePause={live.togglePause}
                onClear={clear}
                liveLimit={live.liveLimit}
                onLiveLimitChange={live.setLiveLimit}
                displayRate={session.displayRate}
                onDisplayRateChange={session.setDisplayRate}
              />
            )}
            {!running && (
              <div className="px-4 py-2 border-b bg-surface-secondary text-xs text-content-secondary" data-testid="subscribe-stopped">
                Stopped. The last messages stay here; press Start to listen again.
              </div>
            )}
            {live.wsError && (
              <div className="px-4 py-2">
                <ErrorAlert message={live.wsError} />
              </div>
            )}
            {feed()}
          </section>
          {selected && (
            <>
              <ResizeHandle {...separatorProps} />
              <aside
                className="bg-surface-secondary flex flex-col overflow-hidden"
                style={{ width: `${rightPanelPct}%` }}
                aria-label="Details panel"
              >
                <UnifiedMessageViewer
                  streamName={null}
                  connectionId={connectionId}
                  selectedMessage={selected}
                  onOpenMappings={handleOpenMappings}
                  onResend={openResend}
                  onReply={openReply}
                  onClose={closeDetails}
                />
              </aside>
            </>
          )}
        </div>
      )}

      {dialog && (
        <CorePublishDialog
          connectionId={connectionId}
          mode={dialog.mode}
          initial={dialog.initial}
          onClose={() => setDialog(null)}
        />
      )}
    </div>
  )
}
