import { useEffect, useMemo, useState } from 'react'
import { Link, useNavigate, useOutletContext, useParams } from 'react-router-dom'
import type { MicroEndpoint } from '@/api/discovery'
import { getErrorMessage } from '@/api/errors'
import Tooltip from '@/components/common/Tooltip'
import { useMicroServices } from '@/contexts/discovery'
import { describePermission, isDenied } from '@/shared/domain/access'
import { getRequestDraft, patchRequestDraft } from '@/stores/requestDraftStore'
import {
  Button,
  CopyButton,
  EmptyState,
  LockClosedIcon,
  QueryErrorState,
  RefreshIcon,
  SearchInput,
  ServicesIcon,
  SkeletonRows,
  Toggle,
  WarningIcon,
} from '@/components/ui'
import { formatTime } from '@/utils/formatters'
import { plural } from '@/utils/plural'
import { AccessDeniedState } from '../common/access/AccessDeniedState'
import type { ConnectionOutletContext } from '../common/ConnectedLayout'
import { ServiceDetail } from './ServiceDetail'
import { ServiceList } from './ServiceList'
import { healthOf, sumWindows, useServiceWindows } from './serviceRates'
import { callDraftPatch, filterServices } from './servicesUtils'

const DEMO_COMMAND = 'nats micro serve demo'

const WHAT_IS_IT =
  'NATS Micro is the service framework built into the NATS clients (Go, JavaScript, Python, Rust and others). ' +
  'Running services answer on the $SRV subjects; natscope asks them every 5 seconds, like nats micro ls.'

function Notice({ children, testId }: { children: React.ReactNode; testId: string }) {
  return (
    <div
      className="mx-4 mt-3 flex items-start gap-2 rounded-md border border-status-warning-border bg-status-warning-bg px-3 py-2 text-xs text-status-warning-text"
      data-testid={testId}
    >
      {children}
    </div>
  )
}

export default function ServicesPage() {
  const { connectionId } = useOutletContext<ConnectionOutletContext>()
  const { name } = useParams<{ name?: string }>()
  const navigate = useNavigate()
  const [autoRefresh, setAutoRefresh] = useState(true)
  const [filter, setFilter] = useState('')
  const [manualRefresh, setManualRefresh] = useState(false)

  const {
    query: { data, error, isLoading, isFetching, dataUpdatedAt },
    recheck,
  } = useMicroServices(connectionId, { autoRefresh })
  const windows = useServiceWindows(connectionId, data, dataUpdatedAt)

  const infoDenied = isDenied(data?.info_access)
  const statsDenied = isDenied(data?.stats_access)
  const statsUnavailable = data?.stats_access?.status === 'unknown'
  const statsShown = !!data && !statsDenied && !statsUnavailable
  const all = useMemo(() => data?.services ?? [], [data?.services])
  const services = useMemo(() => filterServices(all, filter), [all, filter])
  const selected = all.find((s) => s.name === name) ?? null
  const failing = windows ? all.filter((s) => healthOf(sumWindows(windows, (w) => w.service === s.name)) === 'failing').length : 0

  useEffect(() => {
    if (!name && all.length > 0) navigate(`/services/${encodeURIComponent(all[0].name)}`, { replace: true })
  }, [name, all, navigate])

  const refresh = () => {
    setManualRefresh(true)
    void recheck().finally(() => setManualRefresh(false))
  }

  const call = (endpoint: MicroEndpoint) => {
    patchRequestDraft(connectionId, callDraftPatch(endpoint, getRequestDraft(connectionId)))
    navigate('/request')
  }

  const body = () => {
    if (isLoading) return <SkeletonRows count={4} rowClassName="h-10" className="p-6" />
    if (error && !data) return <QueryErrorState error={error} onRetry={refresh} />
    if (!data) return null
    if (infoDenied) {
      const inbox = data.info_access.operation === 'subscribe'
      return (
        <AccessDeniedState
          check={data.info_access}
          title="No access to services"
          description="Finding NATS Micro services needs permission to ask them on the $SRV subjects and to receive their answers on a reply inbox."
          onRetry={refresh}
          retrying={manualRefresh}
          hint={
            inbox ? (
              <>
                If your account receives replies on a private inbox, set its prefix in the{' '}
                <Link to={`/settings/connections/${connectionId}/edit`} className="text-accent hover:text-accent-text">
                  connection settings
                </Link>
                . Otherwise ask the operator of this NATS account to grant it.
              </>
            ) : undefined
          }
        />
      )
    }
    if (all.length === 0) {
      return (
        <div className="flex-1 flex items-center justify-center">
          <EmptyState
            size="lg"
            icon={<ServicesIcon className="w-full h-full" />}
            title="No services found"
            description="Nothing answered on $SRV.INFO within 2 seconds. Services built with NATS Micro show up here while they run."
            action={
              <div className="flex flex-col items-center gap-3">
                <div className="flex items-center gap-1 rounded-md border border-border bg-surface-secondary px-2 py-1">
                  <span className="text-xs text-content-tertiary">Try one:</span>
                  <code className="text-xs text-content-primary">{DEMO_COMMAND}</code>
                  <CopyButton value={DEMO_COMMAND} label="Copy command" size="sm" />
                </div>
                <Button variant="secondary" size="sm" onClick={refresh} loading={manualRefresh}>
                  Look again
                </Button>
              </div>
            }
          />
        </div>
      )
    }
    return (
      <div className="flex-1 min-h-0 flex flex-col">
        {error && (
          <Notice testId="services-refresh-failed">
            <WarningIcon className="w-3.5 h-3.5 mt-0.5 shrink-0" />
            <span>The last refresh failed ({getErrorMessage(error)}); showing the previous answers.</span>
          </Notice>
        )}
        {statsDenied && data.stats_access && (
          <Notice testId="stats-denied">
            <LockClosedIcon className="w-3.5 h-3.5 mt-0.5 shrink-0" />
            <span>
              Statistics are hidden: this NATS user has no permission to <code>{describePermission(data.stats_access)}</code>. Services
              and endpoints are still listed; Refresh checks the permission again.
            </span>
          </Notice>
        )}
        {statsUnavailable && (
          <Notice testId="stats-unavailable">
            <WarningIcon className="w-3.5 h-3.5 mt-0.5 shrink-0" />
            <span>Statistics could not be read this time; they come back with the next refresh that gets them.</span>
          </Notice>
        )}
        <div className="flex-1 min-h-0 flex mt-3 border-t border-border">
          <div className="w-72 shrink-0 border-r border-border flex flex-col min-h-0">
            <div className="p-3 border-b border-border">
              <SearchInput value={filter} onChange={setFilter} placeholder="Filter by name or subject" size="sm" clearable />
            </div>
            <div className="flex-1 min-h-0 overflow-auto">
              {services.length > 0 ? (
                <ServiceList
                  services={services}
                  selected={selected?.name ?? null}
                  windows={windows}
                  statsShown={statsShown}
                  onSelect={(n) => navigate(`/services/${encodeURIComponent(n)}`)}
                />
              ) : (
                <p className="px-4 py-3 text-sm text-content-tertiary">No service matches “{filter}”.</p>
              )}
            </div>
          </div>
          <div className="flex-1 min-w-0 overflow-auto">
            {selected ? (
              <ServiceDetail service={selected} statsShown={statsShown} windows={windows} onCall={call} />
            ) : (
              name && (
                <EmptyState
                  title={`${name} is not answering`}
                  description="It stopped, or it answers too slowly. Pick another service or look again."
                />
              )
            )}
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="flex-1 flex flex-col overflow-hidden bg-surface-primary">
      <div className="px-6 pt-5 pb-4 border-b border-border">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <div className="flex items-center gap-2">
              <h2 className="text-lg font-semibold text-content-primary">Services</h2>
              <Tooltip content={WHAT_IS_IT}>
                <span className="text-xs text-accent cursor-help underline decoration-dotted" tabIndex={0}>
                  What is this?
                </span>
              </Tooltip>
            </div>
            <p className="text-sm text-content-tertiary mt-0.5" data-testid="services-summary">
              {data && !infoDenied && all.length > 0
                ? `${plural(all.length, 'service')} · ${plural(
                    all.reduce((n, s) => n + s.instances.length, 0),
                    'instance',
                  )}${failing > 0 ? ` · ${failing} failing` : ''}`
                : 'Request/reply services built with NATS Micro that answer on this connection.'}
            </p>
          </div>
          {!infoDenied && (
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
            </div>
          )}
        </div>
        {dataUpdatedAt > 0 && !infoDenied && (
          <p className="mt-1 text-xs text-content-muted" data-testid="services-updated">
            Updated {formatTime(dataUpdatedAt)}
            {isFetching && !manualRefresh && ' · refreshing…'}
          </p>
        )}
      </div>
      {body()}
    </div>
  )
}
