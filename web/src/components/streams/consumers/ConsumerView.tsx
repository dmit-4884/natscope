import { Button, CopyButton, PencilIcon, RefreshIcon, InfoIcon } from '@/components/ui'
import Tooltip from '@/components/common/Tooltip'
import JsonViewer from '@/components/common/JsonViewer'
import type { ConsumerInfo } from '@/types/nats'
import { formatDateTime, formatNsDuration } from '@/utils/formatters'
import { consumerConfigToNatsCli } from '../natsCli'
import { StatCard, ConfigRow } from './consumerHelpers'
import { getFilterSubjectsArray } from './consumerUtils'

interface Props {
  consumer: ConsumerInfo
  streamName: string
  isFetching: boolean
  onRefetch: () => void
  onEdit: () => void
  onPause: () => void
  onResume: () => void
  onDelete: () => void
  isResuming: boolean
  /** When set, the server doesn't support pause/resume — buttons are disabled with this tooltip. */
  pauseUnsupportedReason?: string
}

export function ConsumerView({
  consumer,
  streamName,
  isFetching,
  onRefetch,
  onEdit,
  onPause,
  onResume,
  onDelete,
  isResuming,
  pauseUnsupportedReason,
}: Props) {
  const filterSubjects = getFilterSubjectsArray(consumer)

  return (
    <>
      <div className="flex items-center px-4 py-3 border-b bg-surface-secondary shrink-0">
        <div>
          <h3 className="font-semibold text-content-primary">{consumer.name}</h3>
          {consumer.config?.description && (
            <p className="text-sm text-content-tertiary mt-0.5">{consumer.config.description}</p>
          )}
        </div>
      </div>

      <div className="flex-1 overflow-auto p-4">
        <div className="grid grid-cols-1 xl:grid-cols-[1fr,550px] gap-4 items-start">
          <div className="flex flex-col gap-4">
            {/* Stats */}
            <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
              <StatCard
                label="Pending"
                value={consumer.num_pending ?? 0}
                highlight={consumer.num_pending > 0 ? 'warning' : undefined}
                hint="Messages waiting to be delivered to this consumer"
              />
              <StatCard
                label="Ack Pending"
                value={consumer.num_ack_pending ?? 0}
                highlight={consumer.num_ack_pending > 0 ? 'warning' : undefined}
                hint="Messages delivered but not yet acknowledged"
              />
              <StatCard
                label="Delivered"
                value={consumer.delivered?.consumer_seq ?? 0}
                hint="Total messages delivered to this consumer"
              />
              <StatCard
                label="Redelivered"
                value={consumer.num_redelivered ?? 0}
                hint="Messages that were redelivered due to timeout or negative ack"
              />
            </div>

            {/* Configuration */}
            <div className="bg-surface-primary rounded-lg border">
              <div className="px-4 py-3 border-b bg-surface-secondary flex items-center justify-between gap-3">
                <h3 className="font-medium text-content-primary">Configuration</h3>
                <div className="flex items-center gap-1">
                  <Tooltip content="Refresh">
                    <Button variant="ghost" size="sm" onClick={onRefetch} disabled={isFetching} aria-label="Refresh">
                      <RefreshIcon className={`w-4 h-4 ${isFetching ? 'animate-spin' : ''}`} />
                    </Button>
                  </Tooltip>
                  <Button variant="secondary" size="sm" onClick={onEdit}>
                    <PencilIcon className="w-4 h-4 mr-1.5" />
                    Edit
                  </Button>
                </div>
              </div>
              <div className="p-4">
                <div className="mb-4 pb-4 border-b">
                  <div className="flex items-center gap-1.5 text-sm text-content-tertiary mb-2">
                    Filter Subject
                    <Tooltip content="Subject pattern to filter messages. Supports wildcards (* and >)" position="bottom">
                      <InfoIcon className="w-3.5 h-3.5 text-content-muted hover:text-content-tertiary" />
                    </Tooltip>
                  </div>
                  <div className="flex flex-wrap gap-2">
                    {filterSubjects.length === 0 ? (
                      <span className="text-xs text-content-tertiary">— all subjects</span>
                    ) : (
                      filterSubjects.map((subject, idx) => (
                        <span key={idx} className="px-2 py-1 bg-accent-light text-accent-text text-xs font-mono rounded">
                          {subject}
                        </span>
                      ))
                    )}
                  </div>
                </div>

                <div className="grid grid-cols-1 xl:grid-cols-2 gap-x-6 gap-y-1">
                  <ConfigRow
                    label="Deliver Policy"
                    value={consumer.config?.deliver_policy || '-'}
                    hint="Where to start delivering messages: all (from beginning), last (last message), new (only new), by_start_sequence, by_start_time"
                  />
                  <ConfigRow
                    label="Ack Policy"
                    value={consumer.config?.ack_policy || '-'}
                    hint="How messages are acknowledged: explicit (each message), all (cumulative), none (no ack required)"
                  />
                  <ConfigRow
                    label="Ack Wait"
                    value={consumer.config?.ack_wait ? formatNsDuration(consumer.config.ack_wait) : '-'}
                    hint="Time the server waits for an acknowledgment before redelivering the message"
                  />
                  <ConfigRow
                    label="Max Deliver"
                    value={
                      consumer.config?.max_deliver === -1
                        ? 'Unlimited'
                        : String(consumer.config?.max_deliver ?? '-')
                    }
                    hint="Maximum number of delivery attempts for a message before it's considered undeliverable"
                  />
                  <ConfigRow
                    label="Max Ack Pending"
                    value={String(consumer.config?.max_ack_pending ?? '-')}
                    hint="Maximum number of outstanding unacknowledged messages allowed"
                  />
                  <ConfigRow
                    label="Max Waiting"
                    value={String(consumer.config?.max_waiting ?? '-')}
                    hint="Maximum number of waiting pull requests (pull consumers only)"
                  />
                  {consumer.config?.inactive_threshold && (
                    <ConfigRow
                      label="Inactive Threshold"
                      value={formatNsDuration(consumer.config.inactive_threshold)}
                      hint="Time after which an ephemeral consumer is considered inactive and deleted"
                    />
                  )}
                  {consumer.config?.backoff && consumer.config.backoff.length > 0 && (
                    <ConfigRow
                      label="Backoff"
                      value={consumer.config.backoff.map((b: number) => formatNsDuration(b)).join(' → ')}
                      hint="Redelivery backoff intervals for failed messages"
                    />
                  )}
                  {consumer.cluster && (
                    <ConfigRow
                      label="Cluster"
                      value={`${consumer.cluster.name} (leader: ${consumer.cluster.leader})`}
                      hint="NATS cluster information and current leader node"
                    />
                  )}
                  <ConfigRow
                    label="Created"
                    value={consumer.created ? formatDateTime(consumer.created) : '-'}
                    hint="When this consumer was created"
                  />
                </div>
              </div>
            </div>

            <div className="flex gap-2">
              {pauseUnsupportedReason ? (
                <Tooltip content={pauseUnsupportedReason}>
                  <div className="flex gap-2">
                    <Button variant="secondary" size="sm" disabled>
                      Pause
                    </Button>
                    <Button variant="secondary" size="sm" disabled>
                      Resume
                    </Button>
                  </div>
                </Tooltip>
              ) : (
                <>
                  <Button variant="secondary" size="sm" onClick={onPause}>
                    Pause
                  </Button>
                  <Button variant="secondary" size="sm" onClick={onResume} disabled={isResuming}>
                    Resume
                  </Button>
                </>
              )}
              <Button variant="danger" size="sm" onClick={onDelete}>
                Delete
              </Button>
            </div>
          </div>

          <div className="bg-surface-primary rounded-lg border h-fit sticky top-4">
            <div className="flex items-center justify-between px-4 py-3 border-b bg-surface-secondary">
              <h3 className="font-medium text-content-primary">Raw Configuration</h3>
              <div className="flex items-center gap-2">
                <CopyButton
                  value={() => consumerConfigToNatsCli(consumer.name, streamName, consumer.config ?? {})}
                  variant="button"
                  size="sm"
                  label="Copy as nats CLI"
                  successMessage="CLI copied"
                />
                <CopyButton
                  value={JSON.stringify(consumer.raw ?? consumer, null, 2)}
                  variant="button"
                  size="sm"
                  label="Copy JSON"
                />
              </div>
            </div>
            <div className="max-h-[calc(100vh-250px)] overflow-auto">
              <JsonViewer data={consumer.raw ?? consumer} />
            </div>
          </div>
        </div>
      </div>
    </>
  )
}
