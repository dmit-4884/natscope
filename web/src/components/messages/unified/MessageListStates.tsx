import { EmptyState, WarningIcon, InfoIcon } from '@/components/ui'
import type { WsStatus } from './messageListUtils'

export function NoStreamSelected() {
  return <div className="p-4 text-content-tertiary text-sm">Select a stream to view messages</div>
}

export function MessagesLoading() {
  return (
    <div className="flex-1 flex items-center justify-center text-content-tertiary">
      <div className="text-center">
        <svg className="mx-auto h-8 w-8 text-content-muted mb-3 animate-spin" fill="none" viewBox="0 0 24 24">
          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
        </svg>
        <p className="text-sm">Loading messages...</p>
      </div>
    </div>
  )
}

export function RealtimeStatusBar({ status }: { status: WsStatus }) {
  return (
    <div className="px-4 py-2 bg-status-warning-bg text-amber-700 text-sm border-b flex items-center gap-2">
      <span aria-hidden="true" className="w-2 h-2 rounded-full bg-amber-500 animate-pulse" />
      {status === 'connecting' && 'Connecting...'}
      {status === 'reconnecting' && 'Reconnecting...'}
      {status === 'disconnected' && 'Disconnected'}
    </div>
  )
}

const MESSAGES_ICON = (
  <svg className="w-full h-full" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M7 8h10M7 12h4m1 8l-4-4H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-3l-4 4z"
    />
  </svg>
)

export function EmptyMessagesState({ subjectFilter, isRealtime }: { subjectFilter: string; isRealtime: boolean }) {
  const title = subjectFilter
    ? `No messages match “${subjectFilter}”`
    : isRealtime
      ? 'Waiting for messages…'
      : 'No messages found'

  return (
    <div className="flex-1 flex items-center justify-center">
      <EmptyState
        icon={MESSAGES_ICON}
        title={title}
        description={
          subjectFilter
            ? 'Clear the subject filter or widen the pattern.'
            : isRealtime
              ? 'New messages appear here as they are published.'
              : 'This stream has no messages in the selected range.'
        }
      />
    </div>
  )
}

export function WorkQueueWarning({ onSwitchToRealtime }: { onSwitchToRealtime: () => void }) {
  return (
    <div className="px-4 py-2 bg-status-warning-bg border-b border-amber-200 flex items-center gap-2">
      <WarningIcon className="w-4 h-4 text-status-warning-text shrink-0" />
      <span className="text-sm text-amber-800">
        <strong>WorkQueue stream:</strong> Messages are deleted after acknowledgment by consumers. History may be empty
        or limited.{' '}
        <button onClick={onSwitchToRealtime} className="text-amber-700 underline hover:text-amber-900">
          Switch to Realtime
        </button>{' '}
        to see messages as they flow through.
      </span>
    </div>
  )
}

/**
 * WorkQueue live view uses Core NATS subscribe (a JS AckNone consumer would
 * drain the queue); no backlog replay.
 */
export function WorkQueueRealtimeNotice() {
  return (
    <div className="px-4 py-2 bg-accent-light border-b border-blue-200 flex items-start gap-2 text-xs text-blue-800">
      <InfoIcon className="w-4 h-4 mt-0.5 text-accent shrink-0" />
      <span>
        <strong>Core NATS read mode</strong> — on WorkQueue streams the live view subscribes via core pub/sub instead
        of a JetStream consumer, so it never removes messages from the queue. You'll see new messages as they're
        published; backlog isn't replayed.
      </span>
    </div>
  )
}
