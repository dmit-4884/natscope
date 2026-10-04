import Tooltip from '@/components/common/Tooltip'
import { Badge, Button, CopyButton, DataTable, type DataTableColumn } from '@/components/ui'
import type { MicroEndpoint, MicroInstance, MicroService } from '@/api/discovery'
import { formatCount, formatDateTime, formatNanoseconds, formatTimestamp } from '@/utils/formatters'
import { NoAccessValue } from '../common/access/NoAccessValue'

interface Props {
  service: MicroService
  statsDenied: boolean
  onCall: (endpoint: MicroEndpoint) => void
}

const NO_STATS = 'No access to statistics'
const NOT_REPORTED = 'This instance did not report statistics'

function statCell(endpoint: MicroEndpoint, statsDenied: boolean, render: (s: NonNullable<MicroEndpoint['stats']>) => React.ReactNode) {
  if (endpoint.stats) return render(endpoint.stats)
  return <NoAccessValue reason={statsDenied ? NO_STATS : NOT_REPORTED} />
}

export function ServiceDetail({ service, statsDenied, onCall }: Props) {
  const endpointColumns: DataTableColumn<MicroEndpoint>[] = [
    {
      key: 'name',
      header: 'Endpoint',
      width: 'w-[22%]',
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
      width: 'w-[12%]',
      render: (e) => <span className="font-mono text-xs text-content-secondary">{e.queue_group || '—'}</span>,
    },
    {
      key: 'requests',
      header: 'Requests',
      align: 'right',
      width: 'w-[9%]',
      render: (e) => statCell(e, statsDenied, (s) => <span className="tabular-nums">{formatCount(s.num_requests)}</span>),
    },
    {
      key: 'errors',
      header: 'Errors',
      align: 'right',
      width: 'w-[9%]',
      render: (e) =>
        statCell(e, statsDenied, (s) => (
          <span className={`tabular-nums ${s.num_errors > 0 ? 'text-status-error-text font-medium' : ''}`}>
            {formatCount(s.num_errors)}
          </span>
        )),
    },
    {
      key: 'avg',
      header: 'Avg time',
      align: 'right',
      width: 'w-[9%]',
      render: (e) =>
        statCell(e, statsDenied, (s) => (
          <span className="tabular-nums">{s.num_requests > 0 ? formatNanoseconds(s.average_processing_time_ns) : '—'}</span>
        )),
    },
  ]

  const instanceColumns: DataTableColumn<MicroInstance>[] = [
    { key: 'id', header: 'Instance', width: 'w-[30%]', render: (i) => <code className="text-content-primary">{i.id}</code> },
    { key: 'version', header: 'Version', width: 'w-[15%]', render: (i) => <span className="font-mono text-xs">{i.version || '—'}</span> },
    {
      key: 'started',
      header: 'Started',
      width: 'w-[20%]',
      render: (i) =>
        i.started ? (
          <span title={formatDateTime(i.started)}>{formatTimestamp(i.started, 'relative')}</span>
        ) : (
          <NoAccessValue reason={statsDenied ? NO_STATS : NOT_REPORTED} />
        ),
    },
    {
      key: 'metadata',
      header: 'Metadata',
      render: (i) => {
        const entries = Object.entries(i.metadata)
        if (entries.length === 0) return <span className="text-content-muted">—</span>
        return (
          <div className="flex flex-wrap gap-1">
            {entries.map(([k, v]) => (
              <span key={k} className="rounded bg-surface-tertiary px-1.5 py-0.5 text-2xs font-mono text-content-secondary">
                {k}={v}
              </span>
            ))}
          </div>
        )
      },
    },
  ]

  const lastErrors = service.endpoints.filter((e) => e.stats?.last_error)

  return (
    <section aria-label={`Service ${service.name}`} className="px-6 py-5 space-y-5" data-testid="service-detail">
      <div>
        <div className="flex flex-wrap items-center gap-2">
          <h3 className="text-base font-semibold text-content-primary">{service.name}</h3>
          {service.versions.map((v) => (
            <Badge key={v} size="sm">
              v{v}
            </Badge>
          ))}
        </div>
        {service.description && <p className="mt-1 text-sm text-content-secondary">{service.description}</p>}
      </div>

      <div>
        <h4 className="text-xs font-semibold uppercase tracking-wide text-content-tertiary mb-2">
          Endpoints · {service.endpoints.length}
        </h4>
        <div className="rounded-md border border-border overflow-x-auto">
          <DataTable
            columns={endpointColumns}
            items={service.endpoints}
            rowKey={(e) => `${e.name} ${e.subject}`}
            rowActions={(e) => (
              <Button size="sm" variant="secondary" onClick={() => onCall(e)} aria-label={`Call ${e.name}`}>
                Call
              </Button>
            )}
            emptyState={<p className="px-4 py-3 text-sm text-content-tertiary">This service has no endpoints.</p>}
          />
        </div>
        {lastErrors.length > 0 && (
          <ul className="mt-2 space-y-1" aria-label="Last errors">
            {lastErrors.map((e) => (
              <li key={`${e.name} ${e.subject}`} className="text-xs text-status-error-text">
                <span className="font-medium">{e.name}:</span> {e.stats!.last_error}
              </li>
            ))}
          </ul>
        )}
      </div>

      <div>
        <h4 className="text-xs font-semibold uppercase tracking-wide text-content-tertiary mb-2">
          Instances · {service.instances.length}
        </h4>
        <div className="rounded-md border border-border overflow-x-auto">
          <DataTable columns={instanceColumns} items={service.instances} rowKey={(i) => i.id} />
        </div>
      </div>
    </section>
  )
}
