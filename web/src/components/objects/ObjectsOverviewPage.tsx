import { Link, useNavigate, useOutletContext } from 'react-router-dom'
import { useObjectBuckets } from '@/contexts/objects'
import { formatBytes } from '@/utils/formatters'
import { plural } from '@/utils/plural'
import { Badge, EmptyState, PlusIcon, QueryErrorState, SkeletonRows } from '@/components/ui'
import type { ConnectionOutletContext } from '../common/ConnectedLayout'

const OBJECTS_ICON = (
  <svg fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4" />
  </svg>
)

export default function ObjectsOverviewPage() {
  const { connectionId } = useOutletContext<ConnectionOutletContext>()
  const navigate = useNavigate()
  const { data: buckets = [], isLoading, error, refetch } = useObjectBuckets(connectionId)

  return (
    <div className="flex-1 flex flex-col overflow-hidden bg-surface-primary">
      <div className="px-6 pt-5 pb-4 border-b border-border flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold text-content-primary">Object Store</h2>
          <p className="text-sm text-content-tertiary mt-0.5">
            {plural(buckets.length, 'bucket')}
          </p>
        </div>
        <Link
          to="/objects/new"
          className="inline-flex items-center gap-1.5 px-3 py-2 text-sm font-medium text-content-inverse bg-accent hover:bg-accent-hover rounded-lg transition-colors"
        >
          <PlusIcon className="w-4 h-4" />
          New object bucket
        </Link>
      </div>

      <div className="flex-1 overflow-auto p-6">
        {isLoading ? (
          <SkeletonRows count={5} rowClassName="h-16" />
        ) : error ? (
          <QueryErrorState error={error} onRetry={() => refetch()} />
        ) : buckets.length === 0 ? (
          <EmptyState
            icon={OBJECTS_ICON}
            title="No object buckets"
            description="Create a bucket to store files and binary objects on this server."
            action={
              <Link to="/objects/new" className="inline-flex items-center gap-1.5 text-sm text-accent hover:text-accent-text">
                <PlusIcon className="w-4 h-4" />
                Create object bucket
              </Link>
            }
          />
        ) : (
          <ul className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
            {buckets.map((b) => (
              <li key={b.bucket}>
                <button
                  type="button"
                  onClick={() => navigate(`/objects/${encodeURIComponent(b.bucket)}`)}
                  className="w-full text-left border border-border rounded-lg p-4 hover:border-border-strong hover:shadow-card transition-all"
                >
                  <div className="flex items-center justify-between gap-2">
                    <span className="font-medium text-sm text-content-primary truncate font-mono">{b.bucket}</span>
                    <span className="flex items-center gap-1">
                      {b.sealed && <Badge variant="warning" size="sm">sealed</Badge>}
                      <Badge variant="default" size="sm">{b.storage}</Badge>
                    </span>
                  </div>
                  <p className="text-xs text-content-tertiary mt-2">
                    {plural(b.objects, 'object')} · {formatBytes(b.size)}
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
