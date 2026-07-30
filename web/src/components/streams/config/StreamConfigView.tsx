import { Button, CopyButton, CheckIcon, PauseIcon, PencilIcon, RefreshIcon } from '@/components/ui'
import Tooltip from '@/components/common/Tooltip'
import { formatBytes, formatDateTime, formatNsDuration, formatNumber } from '@/utils/formatters'
import type { StreamDetail } from '@/types/nats'
import JsonViewer from '@/components/common/JsonViewer'
import { streamConfigToNatsCli } from '../natsCli'
import { StatCard, ConfigRow, Flag } from './streamConfigHelpers'
import { formatConfigValue, hasMetadata } from './streamConfigUtils'

interface Props {
  streamDetail: StreamDetail
  isFetching: boolean
  onRefetch: () => void
  onStartEdit: () => void
  onPurge: () => void
  onSeal: () => void
  onDelete: () => void
}

export function StreamConfigView({
  streamDetail,
  isFetching,
  onRefetch,
  onStartEdit,
  onPurge,
  onSeal,
  onDelete,
}: Props) {
  const isSealed = streamDetail.config?.sealed
  const showFeatures =
    streamDetail.config.sealed ||
    streamDetail.config.deny_delete ||
    streamDetail.config.deny_purge ||
    streamDetail.config.allow_rollup_hdrs ||
    streamDetail.config.allow_direct ||
    streamDetail.config.mirror_direct ||
    streamDetail.config.allow_msg_ttl ||
    streamDetail.config.allow_atomic

  return (
    <div className="p-4">
      <div className="grid grid-cols-1 xl:grid-cols-[1fr,550px] gap-4 items-start">
        <div className="flex flex-col gap-4">
          {/* Stats cards */}
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
            <StatCard label="Messages" value={formatNumber(streamDetail.messages)} hint="Total number of messages stored in the stream" />
            <StatCard label="Size" value={formatBytes(streamDetail.bytes)} hint="Total size of all messages in the stream" />
            <StatCard label="Consumers" value={String(streamDetail.consumer_count)} hint="Number of consumers attached to this stream" />
            <StatCard label="Subjects" value={String(streamDetail.subjects?.length || 0)} hint="Number of subject patterns this stream listens to" />
          </div>

          {/* Configuration block */}
          <div className="bg-surface-primary rounded-lg border">
            <div className="px-4 py-3 border-b bg-surface-secondary flex items-center justify-between gap-3">
              <h3 className="font-medium text-content-primary">Configuration</h3>
              <div className="flex items-center gap-1">
                <Tooltip content="Refresh">
                  <Button variant="ghost" size="sm" onClick={onRefetch} disabled={isFetching} aria-label="Refresh">
                    <RefreshIcon className={`w-4 h-4 ${isFetching ? 'animate-spin' : ''}`} />
                  </Button>
                </Tooltip>
                <Button variant="secondary" size="sm" onClick={onStartEdit}>
                  <PencilIcon className="w-4 h-4 mr-1.5" />
                  Edit
                </Button>
              </div>
            </div>
            <div className="p-4">
              {/* Subjects */}
              <div className="mb-4 pb-4 border-b">
                <div className="text-xs font-medium text-content-tertiary uppercase tracking-wide mb-2">Subjects</div>
                <div className="flex flex-wrap gap-2">
                  {streamDetail.subjects.map((subject: string, idx: number) => (
                    <span key={idx} className="px-2 py-1 bg-accent-light text-accent-text text-xs font-mono rounded">
                      {subject}
                    </span>
                  ))}
                </div>
              </div>

              {/* General */}
              <div className="mb-4 pb-4 border-b">
                <div className="text-xs font-medium text-content-tertiary uppercase tracking-wide mb-2">General</div>
                <div className="grid grid-cols-1 xl:grid-cols-2 gap-x-6 gap-y-1">
                  <ConfigRow label="Stream Name" value={streamDetail.name} />
                  <ConfigRow
                    label="Created"
                    value={formatDateTime(streamDetail.created)}
                    hint="When this stream was created"
                  />
                  {streamDetail.description && (
                    <ConfigRow
                      label="Description"
                      value={streamDetail.description}
                      hint="Human-readable description of the stream"
                    />
                  )}
                </div>
              </div>

              {/* Storage & Retention */}
              <div className="mb-4 pb-4 border-b">
                <div className="text-xs font-medium text-content-tertiary uppercase tracking-wide mb-2">Storage & Retention</div>
                <div className="grid grid-cols-1 xl:grid-cols-2 gap-x-6 gap-y-1">
                  <ConfigRow
                    label="Retention"
                    value={formatConfigValue(streamDetail.config.retention)}
                    hint="How messages are retained: Limits (by count/size/age), Interest (while consumers exist), Workqueue (each message consumed once)"
                  />
                  <ConfigRow
                    label="Storage"
                    value={formatConfigValue(streamDetail.config.storage || 'File')}
                    hint="Where messages are stored: File (persistent to disk) or Memory (faster but volatile)"
                  />
                  <ConfigRow
                    label="Discard"
                    value={formatConfigValue(streamDetail.config.discard || 'Old')}
                    hint="What to discard when limits are reached: Old (removes oldest) or New (rejects new messages)"
                  />
                  <ConfigRow
                    label="Replicas"
                    value={String(streamDetail.config.num_replicas ?? 1)}
                    hint="Number of replicas for fault tolerance (max 5 in clustered mode)"
                  />
                  {streamDetail.config.compression && (
                    <ConfigRow
                      label="Compression"
                      value={streamDetail.config.compression.toUpperCase()}
                      hint="Compression algorithm used for messages: S2 (Snappy) reduces storage at cost of CPU"
                    />
                  )}
                </div>
              </div>

              {/* Limits */}
              <div className="mb-4 pb-4 border-b">
                <div className="text-xs font-medium text-content-tertiary uppercase tracking-wide mb-2">Limits</div>
                <div className="grid grid-cols-1 xl:grid-cols-2 gap-x-6 gap-y-1">
                  <ConfigRow
                    label="Max Messages"
                    value={streamDetail.config.max_msgs === -1 ? 'Unlimited' : formatNumber(streamDetail.config.max_msgs)}
                    hint="Maximum number of messages to keep in the stream"
                  />
                  <ConfigRow
                    label="Max Bytes"
                    value={streamDetail.config.max_bytes === -1 ? 'Unlimited' : formatBytes(streamDetail.config.max_bytes)}
                    hint="Maximum total size of all messages in the stream"
                  />
                  <ConfigRow label="Max Age" value={formatNsDuration(streamDetail.config.max_age)} hint="Maximum age of messages before automatic deletion" />
                  <ConfigRow
                    label="Max Consumers"
                    value={
                      !streamDetail.config.max_consumers || streamDetail.config.max_consumers === -1
                        ? 'Unlimited'
                        : formatNumber(streamDetail.config.max_consumers)
                    }
                    hint="Maximum number of consumers allowed on this stream"
                  />
                  {streamDetail.config.max_msgs_per_subject !== undefined && streamDetail.config.max_msgs_per_subject > 0 && (
                    <ConfigRow
                      label="Max Msgs/Subject"
                      value={formatNumber(streamDetail.config.max_msgs_per_subject)}
                      hint="Maximum messages per subject (useful for KV-style streams)"
                    />
                  )}
                  {streamDetail.config.max_msg_size !== undefined && streamDetail.config.max_msg_size > 0 && (
                    <ConfigRow
                      label="Max Msg Size"
                      value={formatBytes(streamDetail.config.max_msg_size)}
                      hint="Maximum size of a single message that will be accepted"
                    />
                  )}
                  {streamDetail.config.duplicate_window !== undefined && streamDetail.config.duplicate_window > 0 && (
                    <ConfigRow
                      label="Duplicate Window"
                      value={formatNsDuration(streamDetail.config.duplicate_window)}
                      hint="Time window for detecting duplicate messages via Nats-Msg-Id header"
                    />
                  )}
                </div>
              </div>

              {/* Features */}
              {showFeatures && (
                <div className="mb-4 pb-4 border-b">
                  <div className="text-xs font-medium text-content-tertiary uppercase tracking-wide mb-2">Features</div>
                  <div className="flex flex-wrap gap-2">
                    {streamDetail.config.sealed && <Flag label="Sealed" color="red" hint="Stream is read-only, no new messages can be published" />}
                    {streamDetail.config.deny_delete && <Flag label="Deny Delete" color="red" hint="Individual message deletion is disabled" />}
                    {streamDetail.config.deny_purge && <Flag label="Deny Purge" color="red" hint="Bulk message purging is disabled" />}
                    {streamDetail.config.allow_direct && <Flag label="Direct Get" color="green" hint="Direct message retrieval via GetMsg API is enabled" />}
                    {streamDetail.config.mirror_direct && <Flag label="Mirror Direct" color="green" hint="Direct get operations on mirror stream are enabled" />}
                    {streamDetail.config.allow_rollup_hdrs && <Flag label="Rollup" color="green" hint="Message rollup via Nats-Rollup header is enabled" />}
                    {streamDetail.config.allow_msg_ttl && <Flag label="Per-Msg TTL" color="green" hint="Per-message TTL via Nats-TTL header is enabled" />}
                    {streamDetail.config.allow_atomic && <Flag label="Atomic Publish" color="green" hint="Atomic batch publishing is enabled" />}
                  </div>
                </div>
              )}

              {/* State */}
              {streamDetail.state && (
                <div className={streamDetail.cluster || hasMetadata(streamDetail) ? 'mb-4 pb-4 border-b' : ''}>
                  <div className="text-xs font-medium text-content-tertiary uppercase tracking-wide mb-2">State</div>
                  <div className="grid grid-cols-1 xl:grid-cols-2 gap-x-6 gap-y-1">
                    <ConfigRow label="First Sequence" value={String(streamDetail.state.first_seq)} hint="Sequence number of the first message in the stream" />
                    <ConfigRow label="Last Sequence" value={String(streamDetail.state.last_seq)} hint="Sequence number of the last message in the stream" />
                    <ConfigRow
                      label="First Message"
                      value={streamDetail.state.first_ts > 0 ? formatDateTime(streamDetail.state.first_ts) : '—'}
                      hint="Timestamp of the first message"
                    />
                    <ConfigRow
                      label="Last Message"
                      value={streamDetail.state.last_ts > 0 ? formatDateTime(streamDetail.state.last_ts) : '—'}
                      hint="Timestamp of the last message"
                    />
                  </div>
                </div>
              )}

              {/* Cluster */}
              {streamDetail.cluster && (
                <div className={hasMetadata(streamDetail) ? 'mb-4 pb-4 border-b' : ''}>
                  <div className="text-xs font-medium text-content-tertiary uppercase tracking-wide mb-2">Cluster</div>
                  <div className="grid grid-cols-1 xl:grid-cols-2 gap-x-6 gap-y-1">
                    {streamDetail.cluster.name && <ConfigRow label="Cluster" value={streamDetail.cluster.name} hint="Name of the NATS cluster" />}
                    {streamDetail.cluster.leader && <ConfigRow label="Leader" value={streamDetail.cluster.leader} hint="Current leader node for this stream" />}
                  </div>
                  {streamDetail.cluster.replicas && streamDetail.cluster.replicas.length > 0 && (
                    <div className="mt-3 pt-3 border-t border-gray-100">
                      <div className="text-xs text-content-tertiary mb-2">Replicas</div>
                      <div className="flex flex-wrap gap-2">
                        {streamDetail.cluster.replicas.map((replica, idx) => (
                          <span
                            key={idx}
                            className={`px-2 py-1 text-xs rounded ${
                              replica.current ? 'bg-status-success-bg text-green-700' : 'bg-yellow-50 text-yellow-700'
                            }`}
                            title={replica.current ? 'In sync' : `Lag: ${replica.active}ns`}
                          >
                            {replica.name} {replica.current ? <CheckIcon className="w-3.5 h-3.5" /> : <PauseIcon className="w-3.5 h-3.5" />}
                          </span>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              )}

              {/* Metadata */}
              {hasMetadata(streamDetail) && (
                <div>
                  <div className="text-xs font-medium text-content-tertiary uppercase tracking-wide mb-2">Metadata</div>
                  <div className="grid grid-cols-1 xl:grid-cols-2 gap-x-6 gap-y-1">
                    {Object.entries(streamDetail.config.metadata!).map(([key, value]) => (
                      <ConfigRow key={key} label={key} value={value} />
                    ))}
                  </div>
                </div>
              )}
            </div>
          </div>

          {/* Actions */}
          <div className="flex gap-2">
            <Button variant="secondary" size="sm" onClick={onPurge} disabled={isSealed}>
              Purge
            </Button>
            {!isSealed && (
              <Button variant="secondary" size="sm" onClick={onSeal}>
                Seal
              </Button>
            )}
            <Button variant="danger" size="sm" onClick={onDelete}>
              Delete
            </Button>
          </div>
        </div>

        {/* Raw JSON */}
        <div className="bg-surface-primary rounded-lg border h-fit sticky top-4">
          <div className="flex items-center justify-between px-4 py-3 border-b bg-surface-secondary">
            <h3 className="font-medium text-content-primary">Raw Configuration</h3>
            <div className="flex items-center gap-2">
              <CopyButton
                value={() =>
                  streamConfigToNatsCli({
                    name: streamDetail.name,
                    subjects: streamDetail.subjects ?? [],
                    config: streamDetail.config,
                  })
                }
                variant="button"
                size="sm"
                label="Copy as nats CLI"
                successMessage="CLI copied"
              />
              <CopyButton
                value={JSON.stringify(streamDetail.raw ?? streamDetail, null, 2)}
                variant="button"
                size="sm"
                label="Copy JSON"
              />
            </div>
          </div>
          <div className="max-h-[calc(100vh-250px)] overflow-auto">
            <JsonViewer data={streamDetail.raw ?? streamDetail} />
          </div>
        </div>
      </div>
    </div>
  )
}
