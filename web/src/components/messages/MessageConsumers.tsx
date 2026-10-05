import { useState } from 'react'
import { Link } from 'react-router-dom'
import { getAccessDenial, getErrorMessage } from '@/api/errors'
import { Badge, ChevronDownIcon, ChevronUpIcon, RefreshIcon } from '@/components/ui'
import Tooltip from '@/components/common/Tooltip'
import { useConsumers } from '@/contexts/streams'
import { describePermission } from '@/shared/domain/access'
import { plural } from '@/utils/plural'
import { messageFates, type FateState } from '../streams/consumers/messageFate'

interface Props {
  connectionId: string
  streamName: string
  message: { sequence: number; subject: string; timestamp: number }
}

const STATE_ORDER: FateState[] = [
  'done',
  'done_or_skipped',
  'skipped',
  'delivered',
  'awaiting_ack',
  'awaiting_or_skipped',
  'not_delivered',
]

const SUMMARY: Record<FateState, string> = {
  done: 'done',
  done_or_skipped: 'done or skipped',
  skipped: 'skipped',
  delivered: 'delivered',
  awaiting_ack: 'waiting for ack',
  awaiting_or_skipped: 'waiting for ack or skipped',
  not_delivered: 'not delivered yet',
}

const BADGE: Record<FateState, 'success' | 'warning' | 'primary' | 'default'> = {
  done: 'success',
  done_or_skipped: 'default',
  skipped: 'default',
  delivered: 'success',
  awaiting_ack: 'warning',
  awaiting_or_skipped: 'default',
  not_delivered: 'primary',
}

export function MessageConsumers({ connectionId, streamName, message }: Props) {
  const [open, setOpen] = useState(false)
  const { data: consumers, error, isLoading, isFetching, refetch } = useConsumers(connectionId, streamName)

  const summary = () => {
    if (isLoading) return 'checking…'
    if (error) {
      const denial = getAccessDenial(error)
      return denial ? `hidden, no permission to ${describePermission(denial)}` : `could not be read (${getErrorMessage(error)})`
    }
    if (!consumers || consumers.length === 0) return 'no consumer reads this stream'
    const { fates } = messageFates(consumers, message)
    if (fates.length === 0) return 'none takes this subject'
    return STATE_ORDER.flatMap((state) => {
      const count = fates.filter((f) => f.state === state).length
      return count > 0 ? [`${count} ${SUMMARY[state]}`] : []
    }).join(' · ')
  }

  const result = consumers && consumers.length > 0 ? messageFates(consumers, message) : null
  const expandable = !!result && (result.fates.length > 0 || result.unrelated > 0)

  const toggle = () => {
    if (!open) void refetch()
    setOpen(!open)
  }

  return (
    <div className="px-4 py-2 bg-surface-primary border-b text-xs" data-testid="message-consumers">
      <div className="flex items-center gap-2">
        <button
          type="button"
          onClick={toggle}
          disabled={!expandable}
          aria-expanded={open}
          className="flex items-center gap-1.5 min-w-0 text-left text-content-secondary hover:text-content-primary disabled:hover:text-content-secondary disabled:cursor-default"
        >
          {expandable &&
            (open ? <ChevronUpIcon className="w-3.5 h-3.5 shrink-0" /> : <ChevronDownIcon className="w-3.5 h-3.5 shrink-0" />)}
          <span className="font-medium text-content-tertiary">Consumers:</span>
          <span className="truncate">{summary()}</span>
        </button>
        {open && (
          <Tooltip content="Read the consumer positions again">
            <button
              type="button"
              onClick={() => void refetch()}
              aria-label="Refresh consumer positions"
              className="ml-auto p-1 rounded text-content-muted hover:text-content-primary hover:bg-surface-hover"
            >
              <RefreshIcon className={`w-3.5 h-3.5 ${isFetching ? 'animate-spin' : ''}`} />
            </button>
          </Tooltip>
        )}
      </div>
      {open && result && (
        <div className="mt-2 pl-5">
          <ul className="space-y-1">
            {result.fates.map((fate) => (
              <li key={fate.consumer.name} className="flex items-center gap-2 min-w-0">
                <Link
                  to={`/streams/${encodeURIComponent(streamName)}/consumers?consumer=${encodeURIComponent(fate.consumer.name)}`}
                  className="font-mono text-accent hover:text-accent-text truncate"
                >
                  {fate.consumer.name}
                </Link>
                <Tooltip content={fate.detail}>
                  <Badge variant={BADGE[fate.state]} size="sm">
                    {fate.label}
                    <span className="sr-only">: {fate.detail}</span>
                  </Badge>
                </Tooltip>
              </li>
            ))}
          </ul>
          {result.unrelated > 0 && (
            <p className="mt-1.5 text-content-muted">
              Not for {plural(result.unrelated, 'other consumer')}: the filter leaves {message.subject} out.
            </p>
          )}
        </div>
      )}
    </div>
  )
}
