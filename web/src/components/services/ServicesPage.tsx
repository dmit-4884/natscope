import { useMemo, useState } from 'react'
import { useNavigate, useOutletContext } from 'react-router-dom'
import { useMicroServices } from '@/contexts/discovery'
import { isDenied, describePermission } from '@/shared/domain/access'
import { getRequestDraft, patchRequestDraft } from '@/stores/requestDraftStore'
import type { MicroEndpoint, MicroService } from '@/api/discovery'
import {
  Button,
  DataTable,
  EmptyState,
  LockClosedIcon,
  QueryErrorState,
  RefreshIcon,
  SearchInput,
  ServicesIcon,
  SkeletonRows,
  Toggle,
  type DataTableColumn,
} from '@/components/ui'
import { formatCount, formatNanoseconds, formatTime } from '@/utils/formatters'
import { plural } from '@/utils/plural'
import { AccessDeniedState } from '../common/access/AccessDeniedState'
import { NoAccessValue } from '../common/access/NoAccessValue'
import type { ConnectionOutletContext } from '../common/ConnectedLayout'
import { ServiceDetail } from './ServiceDetail'
import { callDraftPatch, filterServices, serviceTotals } from './servicesUtils'

export default function ServicesPage() {
  const { connectionId } = useOutletContext<ConnectionOutletContext>()
  const navigate = useNavigate()
  const [autoRefresh, setAutoRefresh] = useState(true)
  const [query, setQuery] = useState('')
  const [selectedName, setSelectedName] = useState<string | null>(null)

  const {
    query: { data, error, isLoading, isFetching, dataUpdatedAt },
    recheck,
  } = useMicroServices(connectionId, { autoRefresh })
  const infoDenied = isDenied(data?.info_access)

  const services = useMemo(() => filterServices(data?.services ?? [], query), [data?.services, query])
  const statsDenied = isDenied(data?.stats_access)
  const selected = services.find((s) => s.name === selectedName) ?? services[0] ?? null

  const call = (endpoint: MicroEndpoint) => {
    patchRequestDraft(connectionId, callDraftPatch(endpoint, getRequestDraft(connectionId)))
    navigate('/request')
  }

  const statsCell = (s: MicroService, pick: (t: NonNullable<ReturnType<typeof serviceTotals>>) => React.ReactNode) => {
    const totals = serviceTotals(s)
    if (totals) return pick(totals)
    return <NoAccessValue reason={statsDenied ? 'No access to statistics' : 'No statistics reported'} />
  }

  const columns: DataTableColumn<MicroService>[] = [
    {
      key: 'name',
      header: 'Service',
      width: 'w-[34%]',
      render: (s) => (
        <div className="min-w-0">
          <div className="font-medium text-content-primary truncate">{s.name}</div>
          {s.description && <div className="text-xs text-content-tertiary truncate">{s.description}</div>}
        </div>
      ),
    },
    {
      key: 'version',
      header: 'Version',
      width: 'w-[12%]',
      render: (s) => <span className="font-mono text-xs">{s.versions.join(', ') || '—'}</span>,
    },
    { key: 'instances', header: 'Instances', align: 'right', width: 'w-[10%]', render: (s) => <span className="tabular-nums">{s.instances.length}</span> },
    { key: 'endpoints', header: 'Endpoints', align: 'right', width: 'w-[10%]', render: (s) => <span className="tabular-nums">{s.endpoints.length}</span> },
    {
      key: 'requests',
      header: 'Requests',
      align: 'right',
      width: 'w-[11%]',
      render: (s) => statsCell(s, (t) => <span className="tabular-nums">{formatCount(t.requests)}</span>),
    },
    {
      key: 'errors',
      header: 'Errors',
      align: 'right',
      width: 'w-[10%]',
      render: (s) =>
        statsCell(s, (t) => (
          <span className={`tabular-nums ${t.errors > 0 ? 'text-status-error-text font-medium' : ''}`}>{formatCount(t.errors)}</span>
        )),
    },
    {
      key: 'avg',
      header: 'Avg time',
      align: 'right',
      width: 'w-[13%]',
      render: (s) => statsCell(s, (t) => <span className="tabular-nums">{t.requests > 0 ? formatNanoseconds(t.averageNs) : '—'}</span>),
    },
  ]

  const body = () => {
    if (isLoading) return <SkeletonRows count={4} rowClassName="h-10" className="p-6" />
    if (error) return <QueryErrorState error={error} onRetry={() => void recheck()} />
    if (!data) return null
    if (infoDenied) {
      return (
        <AccessDeniedState
          check={data.info_access}
          title="No access to services"
          description="Finding NATS Micro services needs permission to ask them on the $SRV subjects and to receive their answers on a reply inbox."
          onRetry={() => void recheck()}
          retrying={isFetching}
        />
      )
    }
    if (data.services.length === 0) {
      return (
        <div className="flex-1 flex items-center justify-center">
          <EmptyState
            size="lg"
            icon={<ServicesIcon className="w-full h-full" />}
            title="No services found"
            description="Nothing answered on $SRV.INFO. Services built with NATS Micro (Go, JavaScript, Python, Rust, …) appear here while they run."
            action={
              <Button variant="secondary" size="sm" onClick={() => void recheck()} loading={isFetching}>
                Look again
              </Button>
            }
          />
        </div>
      )
    }
    return (
      <div className="flex-1 min-h-0 overflow-auto">
        {statsDenied && data.stats_access && (
          <div
            className="mx-6 mt-4 flex items-start gap-2 rounded-md border border-status-warning-border bg-status-warning-bg px-3 py-2 text-xs text-status-warning-text"
            data-testid="stats-denied"
          >
            <LockClosedIcon className="w-3.5 h-3.5 mt-0.5 shrink-0" />
            <span>
              Statistics are hidden: this NATS user has no permission to <code>{describePermission(data.stats_access)}</code>.
              Services and endpoints are still listed.
            </span>
          </div>
        )}
        <div className="px-6 pt-4">
          <div className="rounded-md border border-border overflow-x-auto">
            <DataTable
              columns={columns}
              items={services}
              rowKey={(s) => s.name}
              onRowClick={(s) => setSelectedName(s.name)}
              selectedKey={selected?.name}
              emptyState={<p className="px-4 py-3 text-sm text-content-tertiary">No service matches “{query}”.</p>}
            />
          </div>
        </div>
        {selected && <ServiceDetail service={selected} statsDenied={statsDenied} onCall={call} />}
      </div>
    )
  }

  const showControls = !!data && !infoDenied && data.services.length > 0

  return (
    <div className="flex-1 flex flex-col overflow-hidden bg-surface-primary">
      <div className="px-6 pt-5 pb-4 border-b border-border">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <h2 className="text-lg font-semibold text-content-primary">Services</h2>
            <p className="text-sm text-content-tertiary mt-0.5">
              NATS Micro services that answer on this connection.
              {data && !infoDenied && ` ${plural(data.services.length, 'service')} found.`}
            </p>
          </div>
          <div className="flex items-center gap-3">
            {showControls && (
              <div className="w-56">
                <SearchInput value={query} onChange={setQuery} placeholder="Filter services" size="sm" clearable />
              </div>
            )}
            {!infoDenied && (
              <label className="flex items-center gap-2 text-xs text-content-secondary">
                <Toggle checked={autoRefresh} onChange={setAutoRefresh} size="xs" label="Auto-refresh" />
                Auto-refresh
              </label>
            )}
            <Button
              size="sm"
              variant="secondary"
              icon={<RefreshIcon className={`w-3.5 h-3.5 ${isFetching ? 'animate-spin' : ''}`} />}
              onClick={() => void recheck()}
              disabled={isFetching}
            >
              Refresh
            </Button>
          </div>
        </div>
        {dataUpdatedAt > 0 && !infoDenied && (
          <p className="mt-1 text-xs text-content-muted" data-testid="services-updated">
            Updated {formatTime(dataUpdatedAt)}
          </p>
        )}
      </div>
      {body()}
    </div>
  )
}
