import type { SavedConnection } from '@/api/connections'
import { stripErrorCodePrefix } from '@/api/errors'
import { LockClosedIcon, RowActionButton } from '@/components/ui'
import { formatDate } from '@/utils/formatters'
import { AUTH_LABELS } from './connectionFormData'

interface Props {
  connection: SavedConnection
  isActive: boolean
  isPinging?: boolean
  onConnect: () => void
  onPing?: () => void
  onDuplicate?: () => void
  onEdit?: () => void
  onDelete?: () => void
}

/** Render an absolute timestamp as "X seconds/minutes/hours/days ago". */
function relativeTime(ms: number): string {
  if (!ms) return ''
  const diff = Date.now() - ms
  if (diff < 0) return 'just now'
  const sec = Math.round(diff / 1000)
  if (sec < 60) return 'just now'
  const min = Math.round(sec / 60)
  if (min < 60) return `${min} min ago`
  const hour = Math.round(min / 60)
  if (hour < 24) return `${hour}h ago`
  const day = Math.round(hour / 24)
  if (day < 7) return `${day}d ago`
  return formatDate(ms)
}

export function ConnectionCard({
  connection,
  isActive,
  isPinging = false,
  onConnect,
  onPing,
  onDuplicate,
  onEdit,
  onDelete,
}: Props) {
  const authLabel =
    connection.auth && connection.auth.method !== 'none'
      ? connection.auth.username || AUTH_LABELS[connection.auth.method]
      : null

  const meta = connection.meta
  const hasMtls = !!connection.tls?.clientCert && !!connection.tls?.clientKey
  const hasTls =
    !!connection.tls &&
    (!!connection.tls.caCert || hasMtls || connection.tls.skipVerify || connection.tls.tlsFirst)
  const skipVerify = !!connection.tls?.skipVerify

  // Status dot: active (green) takes priority over a failed last probe (red);
  // otherwise neutral gray.
  const dotClasses = isActive
    ? 'bg-green-500 ring-green-200'
    : meta && !meta.lastSuccess
      ? 'bg-red-400 ring-red-100'
      : 'bg-gray-300 ring-gray-100'
  const dotLabel = isActive ? 'Connected' : meta && !meta.lastSuccess ? 'Connection failed' : 'Not connected'

  return (
    <div
      className={`group relative rounded-lg border transition-all ${
        isActive
          ? 'border-green-200 bg-status-success-bg/50 shadow-sm'
          : meta && !meta.lastSuccess
            ? 'border-red-200 bg-status-error-bg/30 hover:shadow-sm'
            : 'border-border bg-surface-primary hover:border-border-strong hover:shadow-sm'
      }`}
    >
      <div className="flex flex-wrap items-center gap-3 px-4 py-3">
        <div role="img" aria-label={dotLabel} className={`w-2.5 h-2.5 rounded-full shrink-0 ring-2 ${dotClasses}`} />

        {/* min-w keeps the name readable; when space runs out the action
            cluster wraps onto its own row instead of crushing the text. */}
        <div className="flex-1 min-w-[200px]">
          <div className="flex items-center gap-2">
            <span className="font-medium text-sm text-content-primary truncate">{connection.name}</span>
            {isActive && (
              <span className="inline-flex items-center gap-1 px-1.5 py-0.5 text-2xs bg-status-success-light text-green-700 rounded-full font-medium">
                <span aria-hidden="true" className="w-1 h-1 rounded-full bg-green-500" />
                Connected
              </span>
            )}
          </div>
          {connection.description && (
            <p
              className="text-xs text-content-tertiary truncate mt-0.5"
              title={connection.description}
            >
              {connection.description}
            </p>
          )}
          <div className="flex items-center gap-2 mt-0.5 flex-wrap">
            <span className="text-xs text-content-muted font-mono truncate">{connection.urls.join(', ')}</span>
            {authLabel && (
              <span className="inline-flex items-center gap-1 px-1.5 py-0.5 text-2xs bg-surface-tertiary text-content-tertiary rounded-full shrink-0">
                <LockClosedIcon className="w-2.5 h-2.5" />
                {authLabel}
              </span>
            )}
            {hasTls && (
              <span
                className={`inline-flex items-center gap-1 px-1.5 py-0.5 text-2xs rounded-full shrink-0 ${
                  skipVerify
                    ? 'bg-status-warning-light text-amber-700'
                    : 'bg-status-success-light text-emerald-700'
                }`}
                title={
                  skipVerify
                    ? 'TLS enabled — certificate verification disabled'
                    : hasMtls
                      ? 'mTLS — client certificate authentication'
                      : 'TLS enabled'
                }
              >
                {hasMtls ? 'mTLS' : skipVerify ? 'TLS skip-verify' : 'TLS'}
              </span>
            )}
            {connection.urls.length > 1 && (
              <span className="inline-flex items-center px-1.5 py-0.5 text-2xs bg-accent-light text-accent rounded-full shrink-0">
                {connection.urls.length} nodes
              </span>
            )}
            {meta?.serverVersion && (
              <span className="text-2xs text-content-muted">v{meta.serverVersion}</span>
            )}
            {meta?.lastSuccess && meta.lastRttMs !== undefined && (
              <span className="text-2xs text-content-muted">{meta.lastRttMs}ms</span>
            )}
            {meta && meta.lastTestedAt > 0 && (
              <span className="text-2xs text-content-muted">
                {meta.lastSuccess ? '' : 'fail · '}
                {relativeTime(meta.lastTestedAt)}
              </span>
            )}
          </div>
          {meta && !meta.lastSuccess && meta.lastError && (
            <p
              className="text-2xs text-status-error-text mt-1 truncate"
              title={meta.lastError}
            >
              {stripErrorCodePrefix(meta.lastError)}
            </p>
          )}
        </div>

        <div className="flex items-center gap-1 shrink-0 ml-auto">
          {!isActive && (
            <button
              onClick={onConnect}
              className="px-3 py-1.5 text-xs font-medium bg-accent text-content-inverse rounded-md hover:bg-accent-hover transition-colors shadow-sm"
            >
              Connect
            </button>
          )}
          {onPing && (
            <RowActionButton
              kind="ping"
              onClick={onPing}
              disabled={isPinging}
              spinning={isPinging}
              label={`Ping ${connection.name}`}
              title="Test this connection"
            />
          )}
          {onDuplicate && (
            <RowActionButton
              kind="duplicate"
              onClick={onDuplicate}
              label={`Duplicate ${connection.name}`}
              title="Duplicate connection"
            />
          )}
          {onEdit && (
            <RowActionButton
              kind="edit"
              onClick={onEdit}
              label={`Edit ${connection.name}`}
              title="Edit connection"
            />
          )}
          {onDelete && (
            <RowActionButton
              kind="delete"
              onClick={onDelete}
              label={`Delete ${connection.name}`}
              title="Delete connection"
            />
          )}
        </div>
      </div>
    </div>
  )
}
