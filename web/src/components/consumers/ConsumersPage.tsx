import { useMemo, useState } from 'react'
import { useNavigate, useOutletContext } from 'react-router-dom'
import { getAccessDenial, getErrorMessage } from '@/api/errors'
import type { UnreadableStream } from '@/api/stats'
import Tooltip from '@/components/common/Tooltip'
import { useConsumersOverview } from '@/contexts/streams'
import { describePermission } from '@/shared/domain/access'
import {
  Button,
  DataTable,
  DownloadIcon,
  EmptyState,
  LockClosedIcon,
  OverflowMenu,
  QueryErrorState,
  RefreshIcon,
  SearchInput,
  SkeletonRows,
  Toggle,
  UsersIcon,
  WarningIcon,
  type DataTableColumn,
} from '@/components/ui'
import { downloadBlob } from '@/utils/download'
import { formatDateTime, formatTime } from '@/utils/formatters'
import { plural } from '@/utils/plural'
import { AccessDeniedState } from '../common/access/AccessDeniedState'
import type { ConnectionOutletContext } from '../common/ConnectedLayout'
import { ConsumerStatus } from '../streams/consumers/ConsumerStatus'
import { statusText } from '../streams/consumers/consumerHealth'
import { getFilterSubjectsArray } from '../streams/consumers/consumerUtils'
import { buildRows, filterRows, formatAgo, rowsToCsv, rowsToJson, type ConsumerRow } from './consumersOverview'

const WHAT_IS_IT =
  'A consumer reads a stream for an application and remembers what it delivered and what clients acknowledged. ' +
  'With Auto-refresh on, natscope reads them every 5 seconds and says in plain words what holds one back.'

function ColumnHint({ label, hint }: { label: string; hint: string }) {
  return (
    <Tooltip content={hint}>
      <span className="cursor-help underline decoration-dotted underline-offset-2" tabIndex={0}>
        {label}
      </span>
    </Tooltip>
  )
}

function Count({ value, max }: { value: number; max?: number }) {
  return (
    <span className="tabular-nums whitespace-nowrap">
      {value.toLocaleString()}
      {max != null && max > 0 && <span className="text-content-muted"> / {max.toLocaleString()}</span>}
    </span>
  )
}

function columns(now: number): DataTableColumn<ConsumerRow>[] {
  return [
    {
      key: 'consumer',
      header: 'Consumer',
      width: 'w-[28%]',
      render: ({ consumer, kind }) => {
        const filters = getFilterSubjectsArray(consumer)
        return (
          <div className="min-w-0 max-w-[20rem]">
            <div className="font-medium text-content-primary truncate" title={consumer.name}>
              {consumer.name}
            </div>
            <div className="text-xs text-content-tertiary truncate">
              <span className="font-mono">{consumer.stream_name}</span>
              {filters.length > 0 && <span className="font-mono"> · {filters.join(', ')}</span>}
              {` · ${kind}`}
              {!consumer.config?.durable_name && ', ephemeral'}
            </div>
          </div>
        )
      },
    },
    {
      key: 'status',
      header: 'Status',
      width: 'w-[24%]',
      render: ({ issues, state }) => <ConsumerStatus issues={issues} state={state} limit={2} />,
    },
    {
      key: 'pending',
      header: <ColumnHint label="Pending" hint="Messages this consumer has not delivered yet" />,
      align: 'right',
      render: ({ consumer }) => <Count value={consumer.num_pending} />,
    },
    {
      key: 'ack',
      header: <ColumnHint label="Waiting for ack" hint="Delivered messages no client has acknowledged yet, out of the most the consumer allows" />,
      align: 'right',
      render: ({ consumer }) => <Count value={consumer.num_ack_pending} max={consumer.config?.max_ack_pending} />,
    },
    {
      key: 'redelivered',
      header: <ColumnHint label="Redelivered" hint="Unacknowledged messages the server had to deliver more than once" />,
      align: 'right',
      render: ({ consumer }) => <Count value={consumer.num_redelivered ?? 0} />,
    },
    {
      key: 'last',
      header: <ColumnHint label="Last delivery" hint="When the consumer last handed a message to a client" />,
      render: ({ consumer }) => {
        const at = consumer.delivered?.last_active
        if (at == null && (consumer.delivered?.consumer_seq ?? 0) > 0) {
          return (
            <span className="text-xs text-content-tertiary" title="The server keeps the delivery time only until it restarts">
              unknown
            </span>
          )
        }
        return (
          <span className="text-xs text-content-secondary whitespace-nowrap" title={at != null ? formatDateTime(at) : undefined}>
            {formatAgo(at, consumer.time_stamp ?? now)}
          </span>
        )
      },
    },
  ]
}

function UnreadableNotice({ streams }: { streams: UnreadableStream[] }) {
  return (
    <div
      className="flex items-start gap-2 rounded-md border border-status-warning-border bg-status-warning-bg px-3 py-2 text-xs text-status-warning-text"
      data-testid="consumers-unreadable"
    >
      <LockClosedIcon className="w-3.5 h-3.5 mt-0.5 shrink-0" />
      <div className="min-w-0">
        <p>Consumers of {plural(streams.length, 'stream')} are not shown:</p>
        <ul className="mt-1 space-y-0.5">
          {streams.map((s) => (
            <li key={s.stream} className="break-all">
              <span className="font-mono font-medium">{s.stream}</span>
              {' — '}
              {s.access ? (
                <>
                  no permission to <code>{describePermission(s.access)}</code>
                </>
              ) : (
                s.error
              )}
            </li>
          ))}
        </ul>
      </div>
    </div>
  )
}

function exportName(extension: string, at: number): string {
  return `consumers-${new Date(at).toISOString().replace(/:/g, '-').slice(0, 19)}.${extension}`
}

export default function ConsumersPage() {
  const { connectionId } = useOutletContext<ConnectionOutletContext>()
  const navigate = useNavigate()
  const [autoRefresh, setAutoRefresh] = useState(true)
  const [query, setQuery] = useState('')
  const [problemsOnly, setProblemsOnly] = useState(false)
  const [manualRefresh, setManualRefresh] = useState(false)

  const { data, error, isLoading, isFetching, dataUpdatedAt, refetch } = useConsumersOverview(connectionId, { autoRefresh })
  const denial = data ? null : getAccessDenial(error)

  const allRows = useMemo(() => (data ? buildRows(data, dataUpdatedAt) : []), [data, dataUpdatedAt])
  const rows = useMemo(() => filterRows(allRows, query, problemsOnly), [allRows, query, problemsOnly])
  const tableColumns = useMemo(() => columns(dataUpdatedAt), [dataUpdatedAt])

  const stuck = allRows.filter((r) => r.state === 'stuck').length
  const warned = allRows.filter((r) => r.state === 'warning').length
  const streamCount = new Set(allRows.map((r) => r.consumer.stream_name)).size

  const refresh = () => {
    setManualRefresh(true)
    void refetch().finally(() => setManualRefresh(false))
  }

  const open = (row: ConsumerRow) =>
    navigate(
      `/streams/${encodeURIComponent(row.consumer.stream_name ?? '')}/consumers?consumer=${encodeURIComponent(row.consumer.name)}`,
    )

  const summary = () => {
    if (!data || allRows.length === 0) return 'Every JetStream consumer on this connection, and what holds it back.'
    const parts = [`${plural(allRows.length, 'consumer')} on ${plural(streamCount, 'stream')}`]
    if (stuck > 0) parts.push(`${stuck} stuck`)
    if (warned > 0) parts.push(`${warned} with warnings`)
    return parts.join(' · ')
  }

  const body = () => {
    if (isLoading) return <SkeletonRows count={6} rowClassName="h-10" className="p-6" />
    if (denial) {
      return (
        <AccessDeniedState
          check={denial}
          title="No access to consumers"
          description="Listing consumers starts with listing the streams, and the server refused that for this NATS user."
          onRetry={refresh}
          retrying={manualRefresh}
        />
      )
    }
    if (error && !data) return <QueryErrorState error={error} onRetry={refresh} />
    if (!data) return null

    return (
      <div className="flex-1 min-h-0 flex flex-col">
        <div className="px-6 pt-3 space-y-2">
          {error && (
            <div
              className="flex items-start gap-2 rounded-md border border-status-warning-border bg-status-warning-bg px-3 py-2 text-xs text-status-warning-text"
              data-testid="consumers-refresh-failed"
            >
              <WarningIcon className="w-3.5 h-3.5 mt-0.5 shrink-0" />
              <span>The last refresh failed ({getErrorMessage(error)}); showing the previous numbers.</span>
            </div>
          )}
          {data.unreadable.length > 0 && <UnreadableNotice streams={data.unreadable} />}
        </div>
        {allRows.length === 0 ? (
          <div className="flex-1 flex items-center justify-center">
            <EmptyState
              size="lg"
              icon={<UsersIcon className="w-full h-full" />}
              title="No consumers yet"
              description="Consumers show up here once an application, natscope or the nats CLI creates one on a stream."
            />
          </div>
        ) : (
          <>
            <div className="px-6 py-3 flex flex-wrap items-center gap-4">
              <div className="w-80 max-w-full">
                <SearchInput
                  value={query}
                  onChange={setQuery}
                  placeholder="Filter by consumer, stream or subject"
                  size="sm"
                  clearable
                />
              </div>
              <label className="flex items-center gap-2 text-xs text-content-secondary">
                <Toggle checked={problemsOnly} onChange={setProblemsOnly} size="xs" label="Problems only" />
                Problems only ({stuck + warned})
              </label>
            </div>
            <div className="flex-1 min-h-0 overflow-auto border-t border-border">
              {rows.length > 0 ? (
                <DataTable columns={tableColumns} items={rows} rowKey={(r) => r.key} onRowClick={open}
                  rowLabel={(r) => `${r.consumer.name} on ${r.consumer.stream_name}: ${statusText(r.issues, r.state)}`}
                  className="min-w-[44rem]"
                />
              ) : (
                <p className="px-6 py-4 text-sm text-content-tertiary">
                  {problemsOnly && !query ? 'No consumer has a problem right now.' : 'No consumer matches the filter.'}
                </p>
              )}
            </div>
          </>
        )}
      </div>
    )
  }

  return (
    <div className="flex-1 flex flex-col overflow-hidden bg-surface-primary">
      <div className="px-6 pt-5 pb-4 border-b border-border">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <div className="flex items-center gap-2">
              <h2 className="text-lg font-semibold text-content-primary">Consumers</h2>
              <Tooltip content={WHAT_IS_IT}>
                <span className="text-xs text-accent cursor-help underline decoration-dotted" tabIndex={0}>
                  What is this?
                </span>
              </Tooltip>
            </div>
            <p className="text-sm text-content-tertiary mt-0.5" data-testid="consumers-summary">
              {summary()}
            </p>
          </div>
          {!denial && (
            <div className="flex items-center gap-3">
              <label className="flex items-center gap-2 text-xs text-content-secondary">
                <Toggle checked={autoRefresh} onChange={setAutoRefresh} size="xs" label="Auto-refresh every 5 seconds" />
                Auto-refresh
              </label>
              <Button
                size="sm"
                variant="secondary"
                icon={<RefreshIcon className={`w-3.5 h-3.5 ${manualRefresh ? 'animate-spin' : ''}`} />}
                onClick={refresh}
                disabled={manualRefresh}
              >
                Refresh
              </Button>
              {rows.length > 0 && (
                <OverflowMenu
                  label="Export consumers"
                  icon={<DownloadIcon className="w-4 h-4" />}
                  items={[
                    {
                      label: 'Export as CSV',
                      onSelect: () => downloadBlob(rowsToCsv(rows), exportName('csv', dataUpdatedAt), 'text/csv'),
                    },
                    {
                      label: 'Export as JSON',
                      onSelect: () => downloadBlob(rowsToJson(rows), exportName('json', dataUpdatedAt), 'application/json'),
                    },
                  ]}
                />
              )}
            </div>
          )}
        </div>
        {dataUpdatedAt > 0 && !denial && (
          <p className="mt-1 text-xs text-content-muted" data-testid="consumers-updated">
            Updated {formatTime(dataUpdatedAt)}
            {isFetching && !manualRefresh && ' · refreshing…'}
          </p>
        )}
      </div>
      {body()}
    </div>
  )
}
