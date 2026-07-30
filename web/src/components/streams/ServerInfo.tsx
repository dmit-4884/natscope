import { getErrorMessage } from '@/api/errors'
import { getServerInfo } from '@/api/stats'
import { useConnectionHealth } from '@/contexts/connection'
import { useConnectionQuery } from '@/hooks/useConnectionQuery'
import { formatBytes, formatNumber } from '@/utils/formatters'
import { Spinner, Alert, CloseIcon, Modal, Badge, CopyButton } from '@/components/ui'

interface Props {
  connectionId: string
  onClose: () => void
}

const STATUS_META: Record<'connected' | 'reconnecting' | 'disconnected', { dot: string; text: string; label: string }> = {
  connected: { dot: 'bg-status-success-border', text: 'text-status-success-text', label: 'connected' },
  reconnecting: { dot: 'bg-status-warning-border', text: 'text-status-warning-text', label: 'reconnecting' },
  disconnected: { dot: 'bg-status-error-border', text: 'text-status-error-text', label: 'disconnected' },
}

function securityLabel(tlsRequired: boolean, authRequired: boolean): string {
  if (tlsRequired && authRequired) return 'Auth · TLS'
  if (authRequired) return 'Auth only'
  if (tlsRequired) return 'TLS only'
  return 'Open'
}

function formatRtt(raw: string): string {
  const m = /^([\d.]+)\s*(ns|µs|us|ms|s)$/.exec(raw)
  if (!m) return raw
  const n = parseFloat(m[1])
  const unit = m[2] === 'us' ? 'µs' : m[2]
  if (unit === 'ns') return `${Math.round(n / 1000)}µs`
  if (unit === 'µs') return `${Math.round(n)}µs`
  if (unit === 'ms') return `${parseFloat(n.toFixed(1))}ms`
  return `${parseFloat(n.toFixed(2))}s`
}

function limitHint(limit: number, format: (n: number) => string): string {
  return limit > 0 ? `/ ${format(limit)}` : '· no limit'
}

export default function ServerInfo({ connectionId, onClose }: Props) {
  const { data, isLoading, error } = useConnectionQuery({
    key: ['serverInfo'],
    connectionId,
    fetcher: (signal) => getServerInfo(connectionId, signal),
  })
  const { status, rtt } = useConnectionHealth(connectionId)

  return (
    <Modal
      isOpen
      onClose={onClose}
      ariaLabel="Server information"
      showCloseButton={false}
      size="lg"
      className="bg-surface-secondary"
    >
      <div className="overflow-y-auto">
        {isLoading && (
          <div className="flex items-center justify-center py-16">
            <Spinner size="md" />
          </div>
        )}

        {error && (
          <div className="p-5">
            <Alert variant="error">
              <div className="text-sm">{getErrorMessage(error)}</div>
            </Alert>
          </div>
        )}

        {data && (
          <div className="p-4 space-y-2">
            {/* Header */}
            <div className="flex items-center justify-between gap-3">
              <div className="flex items-center gap-3 min-w-0">
                <div className="w-9 h-9 bg-accent-muted rounded-xl flex items-center justify-center shrink-0">
                  <svg className="w-5 h-5 text-accent" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01" />
                  </svg>
                </div>
                <div className="min-w-0">
                  <div className="text-sm font-semibold text-content-primary truncate">
                    {data.server_name && data.server_name !== data.server_id ? data.server_name : 'NATS server'}
                  </div>
                  <div className="text-xs text-content-muted font-mono truncate">{data.connected_url}</div>
                </div>
              </div>
              <div className="flex items-center gap-3 shrink-0">
                <div className="flex items-center gap-1.5">
                  <div aria-hidden="true" className={`w-2 h-2 rounded-full ${STATUS_META[status].dot}`} />
                  <span className={`text-xs font-medium ${STATUS_META[status].text}`}>
                    {STATUS_META[status].label}
                    {status === 'connected' && rtt && ` · ${formatRtt(rtt)}`}
                  </span>
                </div>
                <button
                  onClick={onClose}
                  aria-label="Close server info"
                  className="p-1 text-content-muted hover:text-content-secondary transition-colors"
                >
                  <CloseIcon className="w-5 h-5" />
                </button>
              </div>
            </div>

            {/* Top metrics */}
            <div className="grid grid-cols-4 gap-2">
              <MetricCard label="VERSION" value={data.version} />
              <MetricCard label="MAX PAYLOAD" value={formatBytes(data.max_payload)} />
              <MetricCard label="SECURITY" value={securityLabel(data.tls_required, data.auth_required)} />
              <MetricCard label="API LEVEL" value={data.capabilities ? String(data.capabilities.api_level) : '—'} />
            </div>

            {/* Server */}
            <Card title="Server">
              <Row
                label="Server ID"
                value={data.server_id}
                mono
                trailing={<CopyButton value={data.server_id} variant="inline" size="sm" />}
              />
              <Row label="Host" value={`${data.host}:${data.port}`} mono />
              <Row label="Connected URL" value={data.connected_url} mono />
              <Row label="Cluster" value={data.cluster_name || 'Standalone'} mono={Boolean(data.cluster_name)} />
              {data.connect_urls.length > 0 && (
                <Row
                  label={`Cluster endpoints (${data.connect_urls.length})`}
                  value={data.connect_urls.join(', ')}
                  mono
                />
              )}
            </Card>

            {/* JetStream */}
            <Card
              title="JetStream"
              badge={
                <Badge variant={data.jetstream ? 'success' : 'default'} shape="pill" size="sm">
                  {data.jetstream ? 'enabled' : 'disabled'}
                </Badge>
              }
            >
              {data.jetstream && data.js_account ? (
                <>
                  {data.js_account.domain && <Row label="Domain" value={data.js_account.domain} mono />}
                  <div className="grid grid-cols-3 gap-x-4">
                    <Stat
                      label="Streams"
                      value={formatNumber(data.js_account.streams)}
                      secondary={limitHint(data.js_account.stream_limit, formatNumber)}
                    />
                    <Stat
                      label="Consumers"
                      value={formatNumber(data.js_account.consumers)}
                      secondary={limitHint(data.js_account.consumer_limit, formatNumber)}
                    />
                    <Stat label="API calls" value={formatNumber(data.js_account.api_total)} />
                    <Stat
                      label="Memory"
                      value={formatBytes(data.js_account.memory)}
                      secondary={limitHint(data.js_account.memory_limit, formatBytes)}
                      bar={{ used: data.js_account.memory, limit: data.js_account.memory_limit }}
                    />
                    <Stat
                      label="Storage"
                      value={formatBytes(data.js_account.storage)}
                      secondary={limitHint(data.js_account.storage_limit, formatBytes)}
                      bar={{ used: data.js_account.storage, limit: data.js_account.storage_limit }}
                    />
                    <Stat
                      label="API errors"
                      value={formatNumber(data.js_account.api_errors)}
                      warn={data.js_account.api_errors > 0}
                    />
                  </div>
                  {data.capabilities && (
                    <div className="flex items-center justify-between gap-3 py-2">
                      <span className="text-xs text-content-tertiary shrink-0">Capabilities</span>
                      <div className="flex flex-wrap gap-1.5 justify-end">
                        <Badge variant={data.capabilities.consumer_pause ? 'success' : 'default'} shape="pill" size="sm">
                          Consumer pause
                        </Badge>
                        <Badge variant={data.capabilities.message_ttl ? 'success' : 'default'} shape="pill" size="sm">
                          Message TTL
                        </Badge>
                        <Badge variant={data.capabilities.atomic_publish ? 'success' : 'default'} shape="pill" size="sm">
                          Atomic publish
                        </Badge>
                      </div>
                    </div>
                  )}
                </>
              ) : (
                <div className="py-1.5 text-xs text-content-muted">Not available on this server</div>
              )}
            </Card>

            {/* This connection */}
            {data.client_stats && (
              <Card title="This connection">
                <div className="grid grid-cols-3 gap-x-4">
                  <Stat label="Msgs in" value={formatNumber(data.client_stats.in_msgs)} />
                  <Stat label="Msgs out" value={formatNumber(data.client_stats.out_msgs)} />
                  <Stat
                    label="Reconnects"
                    value={formatNumber(data.client_stats.reconnects)}
                    warn={data.client_stats.reconnects > 0}
                  />
                  <Stat label="Data in" value={formatBytes(data.client_stats.in_bytes)} />
                  <Stat label="Data out" value={formatBytes(data.client_stats.out_bytes)} />
                  {status === 'connected' && rtt ? <Stat label="RTT" value={formatRtt(rtt)} /> : <div />}
                </div>
              </Card>
            )}
          </div>
        )}
      </div>
    </Modal>
  )
}

// ── Sub-components ──

function MetricCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="bg-surface-primary rounded-md border border-border px-3 py-2.5">
      <div className="text-2xs uppercase tracking-wider text-content-muted font-medium">{label}</div>
      <div className="text-base font-bold text-content-primary mt-0.5 truncate">{value}</div>
    </div>
  )
}

function Card({ title, badge, children }: {
  title: string
  badge?: React.ReactNode
  children: React.ReactNode
}) {
  return (
    <div className="bg-surface-primary rounded-md border border-border">
      <div className="px-4 py-2 border-b border-gray-100 flex items-center justify-between">
        <span className="text-sm font-medium text-content-primary">{title}</span>
        {badge}
      </div>
      <div className="px-4 py-1 divide-y divide-gray-100">
        {children}
      </div>
    </div>
  )
}

function Row({ label, value, mono, warn, trailing }: {
  label: string
  value: string
  mono?: boolean
  warn?: boolean
  trailing?: React.ReactNode
}) {
  return (
    <div className="flex items-start justify-between gap-3 py-1.5">
      <span className="text-xs text-content-tertiary shrink-0 pt-0.5">{label}</span>
      <span className="flex items-center gap-1 min-w-0">
        <span
          className={`text-xs text-right tabular-nums ${mono ? 'font-mono break-all' : ''} ${
            warn ? 'text-status-error-text font-medium' : 'text-content-primary'
          }`}
        >
          {value}
        </span>
        {trailing}
      </span>
    </div>
  )
}

function Stat({ label, value, secondary, warn, bar }: {
  label: string
  value: string
  secondary?: string
  warn?: boolean
  bar?: { used: number; limit: number }
}) {
  const hasBar = bar !== undefined && bar.limit > 0
  const percentage = hasBar ? Math.min((bar.used / bar.limit) * 100, 100) : 0
  const barColor =
    percentage > 90 ? 'bg-status-error-border' : percentage > 70 ? 'bg-status-warning-border' : 'bg-accent'

  return (
    <div className="py-1.5">
      <div className="text-xs text-content-tertiary">{label}</div>
      <div className={`text-lg font-semibold tabular-nums ${warn ? 'text-status-error-text' : 'text-content-primary'}`}>
        {value}
        {secondary && <span className="text-sm font-normal text-content-muted"> {secondary}</span>}
      </div>
      {hasBar && (
        <div className="h-1 mt-1 bg-surface-tertiary rounded-full overflow-hidden">
          <div className={`h-full rounded-full ${barColor}`} style={{ width: `${Math.max(percentage, 1)}%` }} />
        </div>
      )}
    </div>
  )
}
