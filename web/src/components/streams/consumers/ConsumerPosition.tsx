import { getAccessDenial, getErrorMessage } from '@/api/errors'
import { Button, Spinner } from '@/components/ui'
import Tooltip from '@/components/common/Tooltip'
import { useNextMessage } from '@/contexts/messages'
import { describePermission } from '@/shared/domain/access'
import type { ConsumerInfo, Message } from '@/types/nats'
import { formatDateTime } from '@/utils/formatters'
import { getFilterSubjectsArray } from './consumerUtils'

interface Props {
  connectionId: string
  streamName: string
  consumer: ConsumerInfo
  firstSeq?: number
  onOpenMessage: (message: Message) => void
}

interface SpotProps {
  label: string
  hint: string
  idle: string
  startSeq: number | null
  lastSeq?: number
  connectionId: string
  streamName: string
  subjects: string[]
  onOpenMessage: (message: Message) => void
}

function Spot({ label, hint, idle, startSeq, lastSeq, connectionId, streamName, subjects, onOpenMessage }: SpotProps) {
  const { data: message, error, isLoading } = useNextMessage(connectionId, streamName, startSeq, subjects)

  const value = () => {
    if (startSeq == null) return <span className="text-content-tertiary">{idle}</span>
    if (isLoading) return <Spinner size="sm" />
    if (error) {
      const denial = getAccessDenial(error)
      return (
        <span className="text-status-error-text">
          {denial ? `No permission to ${describePermission(denial)}` : `Could not read it: ${getErrorMessage(error)}`}
        </span>
      )
    }
    if (!message || (lastSeq != null && message.sequence > lastSeq)) {
      return <span className="text-content-tertiary">It is no longer in the stream.</span>
    }
    return (
      <span className="flex items-center gap-2 min-w-0">
        <span className="font-mono font-medium text-content-primary">#{message.sequence}</span>
        <span className="font-mono text-content-secondary truncate" title={message.subject}>
          {message.subject}
        </span>
        <span className="text-content-tertiary whitespace-nowrap">{formatDateTime(message.timestamp)}</span>
        <Button
          size="sm"
          variant="ghost"
          className="ml-auto shrink-0"
          aria-label={`Open message #${message.sequence}`}
          onClick={() => onOpenMessage(message)}
        >
          Open
        </Button>
      </span>
    )
  }

  return (
    <div className="flex items-center gap-3 py-1.5 text-sm min-h-9">
      <Tooltip content={hint}>
        <span className="w-44 shrink-0 text-content-tertiary cursor-help underline decoration-dotted underline-offset-2" tabIndex={0}>
          {label}
        </span>
      </Tooltip>
      <div className="flex-1 min-w-0">{value()}</div>
    </div>
  )
}

export function ConsumerPosition({ connectionId, streamName, consumer, firstSeq, onOpenMessage }: Props) {
  const subjects = getFilterSubjectsArray(consumer)
  const delivered = consumer.delivered?.stream_seq ?? 0
  const ackFloor = consumer.ack_floor?.stream_seq ?? 0

  return (
    <div className="bg-surface-primary rounded-lg border" data-testid="consumer-position">
      <div className="px-4 py-3 border-b bg-surface-secondary">
        <h3 className="font-medium text-content-primary">Where it is</h3>
      </div>
      <div className="px-4 py-2">
        <Spot
          label="Oldest waiting for ack"
          hint="The first delivered message no client has acknowledged; the consumer cannot move its ack floor past it"
          idle="Nothing is waiting for an ack."
          startSeq={consumer.num_ack_pending > 0 ? ackFloor + 1 : null}
          lastSeq={delivered}
          connectionId={connectionId}
          streamName={streamName}
          subjects={subjects}
          onOpenMessage={onOpenMessage}
        />
        <Spot
          label="Next to deliver"
          hint="The next new message for this consumer; redeliveries of unacknowledged messages go out before it"
          idle="Nothing left to deliver."
          startSeq={consumer.num_pending > 0 ? delivered + 1 : null}
          connectionId={connectionId}
          streamName={streamName}
          subjects={subjects}
          onOpenMessage={onOpenMessage}
        />
        <p className="mt-1 text-xs text-content-muted" data-testid="consumer-floor">
          {(consumer.delivered?.consumer_seq ?? 0) === 0
            ? 'Nothing delivered yet'
            : `Delivered up to #${delivered} · ${(consumer.ack_floor?.consumer_seq ?? 0) === 0 ? 'nothing acknowledged yet' : `done up to #${ackFloor}`}`}
        </p>
        {firstSeq != null && (consumer.delivered?.consumer_seq ?? 0) > 0 && delivered + 1 < firstSeq && (
          <p className="mt-1 text-xs text-status-warning-text" data-testid="consumer-lost">
            Messages #{delivered + 1}–#{firstSeq - 1} left the stream before this consumer reached them.
          </p>
        )}
      </div>
    </div>
  )
}
