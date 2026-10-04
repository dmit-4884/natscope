import { useCallback, useMemo, useRef, useState } from 'react'
import { useOutletContext } from 'react-router-dom'
import { useLiveStatsStore } from '@/contexts/live'
import { useMappingItems } from '@/contexts/mappings'
import { useDisplayPreferences, useLivePolicy, useUpdateSettings } from '@/contexts/settings'
import { useResizablePanel } from '@/hooks/useResizablePanel'
import { useSubscribeDraft, withRecentSubjects } from '@/stores/subscribeDraftStore'
import type { SelectedMessage } from '@/types/messages'
import { EmptyState, LockClosedIcon, SignalIcon } from '@/components/ui'
import ErrorAlert from '@/components/ui/ErrorAlert'
import { AccessDeniedState } from '../common/access/AccessDeniedState'
import type { ConnectionOutletContext } from '../common/ConnectedLayout'
import UnifiedMessageViewer from '../messages/UnifiedMessageViewer'
import type { ResendDraft } from '../messages/resend'
import { MessageVirtualTable } from '../messages/unified/MessageVirtualTable'
import type { LiveMessage, WsStatus } from '../messages/unified/messageListUtils'
import { useLiveSubscription } from '../messages/unified/useLiveSubscription'
import { buildSubject } from '../streams/publish/subjectPatternUtils'
import { CorePublishDialog, type CorePublishDraft } from './CorePublishDialog'
import { SubjectBar } from './SubjectBar'
import { SubscribeToolbar } from './SubscribeToolbar'
import { filterReceived } from './subscribeUtils'

const NO_COMPARE = () => false
const NOOP = () => {}

interface PillProps {
  running: boolean
  status: WsStatus
  paused: boolean
  denied: boolean
}

function pillState({ running, status, paused, denied }: PillProps) {
  if (!running) return { label: 'Stopped', dot: 'bg-gray-400', text: 'text-content-tertiary', pulse: false }
  if (denied) return { label: 'No permission', dot: null, text: 'text-content-secondary', pulse: false }
  if (status !== 'connected') return { label: 'Connecting…', dot: 'bg-amber-500', text: 'text-status-warning-text', pulse: true }
  if (paused) return { label: 'Paused', dot: 'bg-amber-500', text: 'text-status-warning-text', pulse: false }
  return { label: 'Live', dot: 'bg-green-500', text: 'text-status-success-text', pulse: true }
}

function StatusPill(props: PillProps) {
  const state = pillState(props)
  return (
    <span
      className={`inline-flex items-center gap-1.5 rounded-full border border-border bg-surface-primary px-2.5 py-1 text-xs font-medium ${state.text}`}
      data-testid="subscribe-status"
    >
      {state.dot ? (
        <span aria-hidden="true" className={`w-2 h-2 rounded-full ${state.dot} ${state.pulse ? 'animate-pulse' : ''}`} />
      ) : (
        <LockClosedIcon className="w-3 h-3" />
      )}
      {state.label}
    </span>
  )
}

export default function SubscribePage() {
  const { connectionId, handleOpenMappings } = useOutletContext<ConnectionOutletContext>()
  const [draft, updateDraft] = useSubscribeDraft(connectionId)
  const [running, setRunning] = useState(false)
  const [query, setQuery] = useState('')
  const [subjectFilter, setSubjectFilter] = useState<string | null>(null)
  const [selected, setSelected] = useState<SelectedMessage | null>(null)
  const [dialog, setDialog] = useState<{ mode: 'resend' | 'reply'; initial: CorePublishDraft } | null>(null)

  const liveSettings = useLivePolicy()
  const updateSettings = useUpdateSettings()
  const display = useDisplayPreferences()
  const stats = useLiveStatsStore((s) => s.stats)
  const { data: mappings = [] } = useMappingItems()
  const { rightPanelPct, containerRef, separatorProps } = useResizablePanel()
  const autoScrollRef = useRef(true)

  const live = useLiveSubscription({
    connectionId,
    streamName: null,
    subjects: running ? draft.subjects : undefined,
    enabled: running,
    maxDisplayRate: liveSettings.maxDisplayRate,
    initialLimit: 100,
  })

  const suggestions = useMemo(
    () => [...new Set([...draft.recentSubjects, ...mappings.map((m) => m.pattern)])],
    [draft.recentSubjects, mappings],
  )

  const visible = useMemo(
    () => filterReceived(live.liveMessages, query, subjectFilter),
    [live.liveMessages, query, subjectFilter],
  )

  const allDenied =
    running && draft.subjects.length > 0 && draft.subjects.every((s) => live.deniedSubjects.includes(s))

  const start = (subjects: string[]) => {
    updateDraft({ recentSubjects: withRecentSubjects(draft.recentSubjects, subjects) })
    setRunning(true)
  }

  const clear = () => {
    live.clearMessages()
    setSelected(null)
    setSubjectFilter(null)
  }

  const selectLive = useCallback((msg: LiveMessage) => {
    setSelected({
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
      reply: msg.reply,
      isLive: true,
    })
  }, [])

  const openResend = useCallback((d: ResendDraft) => {
    setDialog({
      mode: 'resend',
      initial: { subject: buildSubject(d.pattern, d.wildcards), payload: d.messageJson, headers: d.headers },
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
          check={{ status: 'denied', operation: 'subscribe', subject: live.deniedSubjects[0] }}
          title="No permission to subscribe"
          description="The server refused every subject in this subscription for your NATS user."
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
          <EmptyState title="No received message matches" description="Clear the search or pick another subject." />
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
        rowHeight={isCompact ? 36 : 52}
        cellPadding={isCompact ? 'px-3 py-1' : 'px-3 py-2'}
        streamName=""
        connectionId={connectionId}
        timestampFormat={display.timestampFormat as 'relative' | 'absolute' | 'iso'}
        showSequence={false}
        autoScrollRef={autoScrollRef}
        onSelectHistory={NOOP}
        onSelectLive={selectLive}
        onCompareSelect={NOOP}
      />
    )
  }

  return (
    <div className="flex-1 flex flex-col overflow-hidden bg-surface-primary">
      <div className="px-6 pt-5 pb-4 border-b border-border">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h2 className="text-lg font-semibold text-content-primary">Subscribe</h2>
            <p className="text-sm text-content-tertiary mt-0.5">
              Watch any subject over core NATS. Nothing is stored: the feed starts when you subscribe.
            </p>
          </div>
          <StatusPill running={running} status={live.wsStatus} paused={live.isPaused} denied={allDenied} />
        </div>
        <SubjectBar
          subjects={draft.subjects}
          onSubjectsChange={(subjects) => updateDraft({ subjects })}
          deniedSubjects={running ? live.deniedSubjects : []}
          suggestions={suggestions}
          recentSubjects={draft.recentSubjects}
          running={running}
          onStart={start}
          onStop={() => setRunning(false)}
        />
        {draft.subjects.includes('>') && (
          <p className="mt-2 text-xs text-content-tertiary">
            System subjects ($JS, $SYS, _INBOX) show up only when you subscribe to them by name.
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
                msgPerSecond={stats?.msgPerSecond}
                running={running}
                isPaused={live.isPaused}
                onTogglePause={live.togglePause}
                onClear={clear}
                liveLimit={live.liveLimit}
                onLiveLimitChange={live.setLiveLimit}
                maxDisplayRate={liveSettings.maxDisplayRate ?? 0}
                onMaxDisplayRateChange={(rate) => updateSettings.mutate({ live: { maxDisplayRate: rate } })}
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
          <div
            {...separatorProps}
            className="flex-shrink-0 cursor-col-resize group flex items-stretch focus:outline-none focus-visible:ring-2 focus-visible:ring-border-focus"
            style={{ padding: '0 2px' }}
          >
            <div className="w-px bg-surface-hover group-hover:bg-blue-400 group-active:bg-blue-500 group-focus-visible:bg-blue-500 transition-colors" />
          </div>
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
            />
          </aside>
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
