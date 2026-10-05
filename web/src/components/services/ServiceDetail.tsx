import { useState } from 'react'
import type { MicroEndpoint, MicroInstance, MicroService } from '@/api/discovery'
import Tooltip from '@/components/common/Tooltip'
import { Badge, Button, CopyButton, DataTable, type DataTableColumn } from '@/components/ui'
import { formatCount, formatDateTime, formatNanoseconds, formatTimestamp } from '@/utils/formatters'
import { HealthBadge } from './HealthBadge'
import { InstanceJsonModal } from './InstanceJsonModal'
import { healthOf, sumWindows, type EndpointWindow, type WindowTotal } from './serviceRates'
import {
  formatErrorShare,
  formatRequestRate,
  formatWindowAverage,
  hasWildcard,
  serviceTotals,
  shortInstanceId,
} from './servicesUtils'

interface Props {
  service: MicroService
  statsShown: boolean
  windows: EndpointWindow[] | null
  onCall: (endpoint: MicroEndpoint) => void
}

const MEASURING = 'Measuring: rates appear after the next refresh'

function MetadataChips({ metadata }: { metadata: Record<string, string> }) {
  const entries = Object.entries(metadata)
  if (entries.length === 0) return null
  return (
    <div className="mt-1 flex flex-wrap gap-1">
      {entries.map(([k, v]) => (
        <span key={k} className="rounded bg-surface-tertiary px-1.5 py-0.5 text-2xs font-mono text-content-secondary">
          {k}={v}
        </span>
      ))}
    </div>
  )
}

function WindowCell({ window, render }: { window: WindowTotal | undefined; render: (w: WindowTotal) => string }) {
  if (!window) {
    return (
      <Tooltip content={MEASURING}>
        <span className="text-content-muted">…</span>
      </Tooltip>
    )
  }
  return <span className="tabular-nums">{render(window)}</span>
}

function Stat({ label, value, hint }: { label: string; value: string; hint?: string }) {
  const body = (
    <div className="min-w-[7rem]">
      <div className="text-2xs font-semibold uppercase tracking-wide text-content-tertiary">{label}</div>
      <div className="mt-0.5 text-base font-semibold text-content-primary tabular-nums">{value}</div>
    </div>
  )
  return hint ? <Tooltip content={hint}>{body}</Tooltip> : body
}

function statColumns<T>(statsShown: boolean, windowOf: (item: T) => WindowTotal | undefined): DataTableColumn<T>[] {
  if (!statsShown) return []
  return [
    { key: 'rate', header: 'Req/s', align: 'right', width: 'w-[10%]', render: (item) => <WindowCell window={windowOf(item)} render={formatRequestRate} /> },
    {
      key: 'errors',
      header: 'Errors',
      align: 'right',
      width: 'w-[9%]',
      render: (item) => {
        const w = windowOf(item)
        return (
          <span className={w && w.errors > 0 ? 'text-status-error-text font-medium' : ''}>
            <WindowCell window={w} render={formatErrorShare} />
          </span>
        )
      },
    },
    { key: 'avg', header: 'Avg time', align: 'right', width: 'w-[10%]', render: (item) => <WindowCell window={windowOf(item)} render={formatWindowAverage} /> },
  ]
}

export function ServiceDetail({ service, statsShown, windows, onCall }: Props) {
  const [inspected, setInspected] = useState<MicroInstance | null>(null)
  const of = (match: (w: EndpointWindow) => boolean) =>
    windows ? sumWindows(windows, (w) => w.service === service.name && match(w)) : undefined
  const serviceWindow = of(() => true)
  const totals = serviceTotals(service)

  const endpointColumns: DataTableColumn<MicroEndpoint>[] = [
    {
      key: 'name',
      header: 'Endpoint',
      width: 'w-[24%]',
      render: (e) => (
        <div className="min-w-0">
          <div className="font-medium text-content-primary truncate">{e.name}</div>
          {e.proto_method && (
            <Tooltip content={`${e.proto_method.input_type} → ${e.proto_method.output_type}`}>
              <span className="mt-0.5 inline-flex">
                <Badge variant="primary" size="sm">
                  {e.proto_method.service.split('.').pop()}.{e.proto_method.method}
                </Badge>
              </span>
            </Tooltip>
          )}
          <MetadataChips metadata={e.metadata} />
        </div>
      ),
    },
    {
      key: 'subject',
      header: 'Subject',
      width: 'w-[26%]',
      render: (e) => (
        <div className="flex items-center gap-1 min-w-0">
          <code className="truncate text-content-primary" title={e.subject}>
            {e.subject}
          </code>
          <CopyButton value={e.subject} label={`Copy ${e.subject}`} size="sm" />
        </div>
      ),
    },
    {
      key: 'queue',
      header: 'Queue group',
      width: 'w-[11%]',
      render: (e) =>
        e.queue_group ? (
          <span className="font-mono text-xs text-content-secondary">{e.queue_group}</span>
        ) : (
          <Tooltip content="No queue group: every instance answers each request">
            <span className="text-xs text-content-muted">none</span>
          </Tooltip>
        ),
    },
    ...statColumns<MicroEndpoint>(statsShown, (e) => of((w) => w.endpoint === e.name && w.subject === e.subject)),
  ]

  const startedColumn: DataTableColumn<MicroInstance> = {
    key: 'started',
    header: 'Up since',
    width: 'w-[12%]',
    render: (i) =>
      i.started ? <span title={formatDateTime(i.started)}>{formatTimestamp(i.started, 'relative')}</span> : <span className="text-content-muted">—</span>,
  }

  const instanceColumns: DataTableColumn<MicroInstance>[] = [
    {
      key: 'id',
      header: 'Instance',
      width: 'w-[18%]',
      render: (i) => (
        <span className="inline-flex items-center gap-1">
          <code className="text-content-primary" title={i.id}>
            {shortInstanceId(i.id)}
          </code>
          <CopyButton value={i.id} label={`Copy instance ID ${i.id}`} size="sm" />
        </span>
      ),
    },
    { key: 'version', header: 'Version', width: 'w-[11%]', render: (i) => <span className="font-mono text-xs">{i.version || '—'}</span> },
    ...statColumns<MicroInstance>(statsShown, (i) => of((w) => w.instance === i.id)),
    ...(statsShown ? [startedColumn] : []),
    {
      key: 'rtt',
      header: 'RTT',
      align: 'right',
      width: 'w-[9%]',
      render: (i) =>
        i.rtt_ns !== undefined ? (
          <Tooltip content="How long the instance took to answer the last discovery request">
            <span className="tabular-nums">{formatNanoseconds(i.rtt_ns)}</span>
          </Tooltip>
        ) : (
          <span className="text-content-muted">—</span>
        ),
    },
    { key: 'metadata', header: 'Metadata', render: (i) => <MetadataChips metadata={i.metadata} /> },
  ]

  const lastErrors = service.instances.flatMap((instance) =>
    instance.endpoints
      .filter((e) => e.stats?.last_error)
      .map((e) => ({ instance, endpoint: e, error: e.stats?.last_error ?? '' })),
  )

  return (
    <section aria-label={`Service ${service.name}`} className="px-6 py-5 space-y-6" data-testid="service-detail">
      <div>
        <div className="flex flex-wrap items-center gap-2">
          <h3 className="text-base font-semibold text-content-primary">{service.name}</h3>
          {service.versions.map((v) => (
            <Badge key={v} size="sm">
              v{v}
            </Badge>
          ))}
          {statsShown && <HealthBadge health={healthOf(serviceWindow)} />}
        </div>
        {service.description && <p className="mt-1 text-sm text-content-secondary">{service.description}</p>}
        {statsShown && (
          <div className="mt-4 flex flex-wrap gap-x-8 gap-y-3" data-testid="service-stats">
            <Stat
              label="Requests"
              value={serviceWindow ? formatRequestRate(serviceWindow) : '…'}
              hint={totals ? `${formatCount(totals.requests)} since the instances started` : MEASURING}
            />
            <Stat
              label="Errors"
              value={serviceWindow ? formatErrorShare(serviceWindow) : '…'}
              hint={totals ? `${formatCount(totals.errors)} since the instances started` : MEASURING}
            />
            <Stat label="Avg time" value={serviceWindow ? formatWindowAverage(serviceWindow) : '…'} />
            <Stat label="Instances" value={String(service.instances.length)} />
          </div>
        )}
      </div>

      <div>
        <h4 className="text-xs font-semibold uppercase tracking-wide text-content-tertiary mb-2">Endpoints · {service.endpoints.length}</h4>
        <div className="rounded-md border border-border overflow-x-auto">
          <DataTable
            columns={endpointColumns}
            items={service.endpoints}
            rowKey={(e) => `${e.name} ${e.subject}`}
            rowActions={(e) => (
              <Tooltip
                content={
                  hasWildcard(e.subject)
                    ? 'Opens Request / Reply; fill in the wildcard tokens there'
                    : 'Send a request to this endpoint in Request / Reply'
                }
              >
                <Button size="sm" variant="secondary" onClick={() => onCall(e)} aria-label={`Call ${e.name}`}>
                  Call
                </Button>
              </Tooltip>
            )}
            emptyState={<p className="px-4 py-3 text-sm text-content-tertiary">This service has no endpoints.</p>}
          />
        </div>
        {lastErrors.length > 0 && (
          <div className="mt-3">
            <h5 className="text-2xs font-semibold uppercase tracking-wide text-content-tertiary mb-1">Last errors</h5>
            <ul className="space-y-1" aria-label="Last errors">
              {lastErrors.map(({ instance, endpoint, error }) => (
                <li key={`${instance.id} ${endpoint.name} ${endpoint.subject}`} className="text-xs text-content-secondary">
                  <span className="font-medium text-content-primary">{endpoint.name}</span>
                  <span className="text-content-tertiary">
                    {' '}
                    on {shortInstanceId(instance.id)}
                    {instance.version && ` (v${instance.version})`}:{' '}
                  </span>
                  <span className="font-mono text-status-error-text">{error}</span>
                </li>
              ))}
            </ul>
          </div>
        )}
      </div>

      <div>
        <h4 className="text-xs font-semibold uppercase tracking-wide text-content-tertiary mb-2">Instances · {service.instances.length}</h4>
        <div className="rounded-md border border-border overflow-x-auto">
          <DataTable
            columns={instanceColumns}
            items={service.instances}
            rowKey={(i) => i.id}
            rowActions={(i) => (
              <Button size="sm" variant="ghost" onClick={() => setInspected(i)} aria-label={`Show the raw replies of ${i.id}`}>
                Raw JSON
              </Button>
            )}
          />
        </div>
      </div>

      {inspected && <InstanceJsonModal serviceName={service.name} instance={inspected} onClose={() => setInspected(null)} />}
    </section>
  )
}
