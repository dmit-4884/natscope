import { Link, useNavigate, useOutletContext } from 'react-router-dom'
import { useKVBuckets } from '@/contexts/kv'
import { formatBytes } from '@/utils/formatters'
import { plural } from '@/utils/plural'
import { Badge, EmptyState, PlusIcon, QueryErrorState, SkeletonRows } from '@/components/ui'
import type { ConnectionOutletContext } from '../common/ConnectedLayout'

const KV_ICON = (
  <svg fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4" />
  </svg>
)

export default function KVOverviewPage() {
  const { connectionId } = useOutletContext<ConnectionOutletContext>()
  const navigate = useNavigate()
  const { data: buckets = [], isLoading, error, refetch } = useKVBuckets(connectionId)

  return (
    <div className="flex-1 flex flex-col overflow-hidden bg-surface-primary">
      <div className="px-6 pt-5 pb-4 border-b border-border flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold text-content-primary">KV Stores</h2>
          <p className="text-sm text-content-tertiary mt-0.5">
            {plural(buckets.length, 'bucket')}
          </p>
        </div>
        <Link
          to="/kv/new"
          className="inline-flex items-center gap-1.5 px-3 py-2 text-sm font-medium text-content-inverse bg-accent hover:bg-accent-hover rounded-lg transition-colors"
        >
          <PlusIcon className="w-4 h-4" />
          New KV bucket
        </Link>
      </div>

      <div className="flex-1 overflow-auto p-6">
        {isLoading ? (
          <SkeletonRows count={5} rowClassName="h-16" />
        ) : error ? (
          <QueryErrorState error={error} onRetry={() => refetch()} />
        ) : buckets.length === 0 ? (
          <EmptyState
            icon={KV_ICON}
            title="No KV buckets"
            description="Create a bucket to store key-value data on this server."
            action={
              <Link to="/kv/new" className="inline-flex items-center gap-1.5 text-sm text-accent hover:text-accent-text">
                <PlusIcon className="w-4 h-4" />
                Create KV bucket
              </Link>
            }
          />
        ) : (
          <ul className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
            {buckets.map((b) => (
              <li key={b.bucket}>
                <button
                  type="button"
                  onClick={() => navigate(`/kv/${encodeURIComponent(b.bucket)}`)}
                  className="w-full text-left border border-border rounded-lg p-4 hover:border-border-strong hover:shadow-card transition-all"
                >
                  <div className="flex items-center justify-between gap-2">
                    <span className="font-medium text-sm text-content-primary truncate font-mono">{b.bucket}</span>
                    <Badge variant="default" size="sm">{b.storage}</Badge>
                  </div>
                  <p className="text-xs text-content-tertiary mt-2">
                    {plural(b.values, 'value')} · {formatBytes(b.bytes)} · history {b.history}
                  </p>
                  {b.description && (
                    <p className="text-xs text-content-muted mt-1 truncate">{b.description}</p>
                  )}
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  )
}
