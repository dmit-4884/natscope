import { Button } from '@/components/ui'
import Tooltip from '@/components/common/Tooltip'
import type { ConsumerInfo, PriorityPolicy } from '@/types/nats'
import { formatDateTime, formatNsDuration } from '@/utils/formatters'
import { ConfigRow } from './consumerHelpers'

const PRIORITY_POLICY_LABELS: Record<PriorityPolicy, string> = {
  none: 'None',
  pinned_client: 'Pinned client',
  overflow: 'Overflow',
  prioritized: 'Prioritized',
}

interface Props {
  consumer: ConsumerInfo
  onUnpin: (group: string) => void
  isUnpinning: boolean
  /** When set, the server doesn't support priority groups — unpin is disabled with this tooltip. */
  unpinUnsupportedReason?: string
}

export function ConsumerPriority({ consumer, onUnpin, isUnpinning, unpinUnsupportedReason }: Props) {
  const policy = consumer.config?.priority_policy ?? 'none'
  if (policy === 'none') return null

  const groups = consumer.config?.priority_groups ?? []
  const states = consumer.priority_groups ?? []

  return (
    <div className="mt-4 pt-4 border-t">
      <div className="text-xs font-medium text-content-tertiary uppercase tracking-wide mb-2">Priority Groups</div>
      <div className="grid grid-cols-[repeat(auto-fit,minmax(min(100%,340px),1fr))] gap-x-6 gap-y-1">
        <ConfigRow
          label="Priority Policy"
          value={PRIORITY_POLICY_LABELS[policy]}
          hint="How pull requests in the priority groups share messages: pinned client, overflow or prioritized"
        />
        {policy === 'pinned_client' && consumer.config?.priority_timeout ? (
          <ConfigRow
            label="Pinned TTL"
            value={formatNsDuration(consumer.config.priority_timeout)}
            hint="How long the pinned client may stay idle before another client is pinned"
          />
        ) : null}
      </div>
      <div className="flex flex-wrap gap-2 mt-2">
        {groups.map((group) => (
          <span key={group} className="px-2 py-1 bg-accent-light text-accent-text text-xs font-mono rounded">
            {group}
          </span>
        ))}
      </div>
      {policy === 'pinned_client' && states.length > 0 && (
        <ul className="mt-3 space-y-1.5">
          {states.map((state) => (
            <li key={state.group} className="flex items-center justify-between gap-3 text-sm">
              <span className="min-w-0 truncate">
                <span className="font-mono text-content-secondary">{state.group}</span>
                {state.pinned_client_id ? (
                  <>
                    <span className="text-content-tertiary"> pinned to </span>
                    <code className="text-content-primary">{state.pinned_client_id}</code>
                    {state.pinned_ts ? (
                      <span className="text-content-tertiary"> since {formatDateTime(state.pinned_ts)}</span>
                    ) : null}
                  </>
                ) : (
                  <span className="ml-2 text-content-tertiary">No client pinned</span>
                )}
              </span>
              {state.pinned_client_id && (
                <Tooltip content={unpinUnsupportedReason ?? 'Release the pinned client so the next pull request is pinned'}>
                  <Button
                    variant="secondary"
                    size="sm"
                    aria-label={`Unpin ${state.group}`}
                    onClick={() => onUnpin(state.group)}
                    disabled={!!unpinUnsupportedReason || isUnpinning}
                  >
                    Unpin
                  </Button>
                </Tooltip>
              )}
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
