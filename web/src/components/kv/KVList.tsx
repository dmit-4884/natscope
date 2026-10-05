import { useMemo } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useConnectionPolicy } from '@/contexts/connection'
import { useStreamNames } from '@/contexts/streams'
import ErrorAlert from '@/components/ui/ErrorAlert'
import { EmptyState, PlusIcon, RefreshIcon, SkeletonRows } from '@/components/ui'
import { SidebarResourceList } from '@/components/common/sidebar/SidebarResourceList'
import { getErrorMessage } from '@/api/errors'

const KV_ICON = (
  <svg fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4" />
  </svg>
)

const KV_STREAM_PREFIX = 'KV_'

interface KVListProps {
  connectionId: string
}

const bucketHref = (name: string) => `/kv/${encodeURIComponent(name)}`

export default function KVList({ connectionId }: KVListProps) {
  const { readOnly } = useConnectionPolicy()
  const { bucketName: selectedBucket } = useParams()
  const { data: streamNames, isLoading, isFetching, error, refetch } = useStreamNames(connectionId)

  // Filter to only show KV streams (streams starting with KV_), without the prefix
  const buckets = useMemo(
    () =>
      (streamNames ?? [])
        .filter((name) => name.startsWith(KV_STREAM_PREFIX))
        .map((name) => name.slice(KV_STREAM_PREFIX.length)),
    [streamNames],
  )

  if (isLoading) {
    return <SkeletonRows count={5} rowClassName="h-9" className="p-2" />
  }

  if (error) {
    return (
      <div className="p-4">
        <ErrorAlert message={`Failed to load KV stores: ${getErrorMessage(error)}`} />
      </div>
    )
  }

  if (buckets.length === 0) {
    return (
      <EmptyState
        size="sm"
        icon={KV_ICON}
        title="No KV stores"
        description="Create your first key-value bucket to start storing data."
        action={
          <div className="flex items-center justify-center gap-3">
            <button
              onClick={() => refetch()}
              disabled={isFetching}
              className="inline-flex items-center gap-1.5 text-sm text-content-tertiary hover:text-gray-700"
            >
              <RefreshIcon className={`w-4 h-4 ${isFetching ? 'animate-spin' : ''}`} />
              Refresh
            </button>
            {!readOnly && (
              <Link
                to={`/kv/new`}
                className="inline-flex items-center gap-1.5 text-sm text-accent hover:text-accent-text"
              >
                <PlusIcon className="w-4 h-4" />
                Create KV store
              </Link>
            )}
          </div>
        }
      />
    )
  }

  return (
    <div>
      {!readOnly && (
        <Link
          to={`/kv/new`}
          className="px-3 py-2.5 flex items-center gap-2 border-b border-border bg-surface-secondary hover:bg-surface-tertiary transition-colors"
        >
          <span className="text-content-muted">
            <PlusIcon className="w-4 h-4" />
          </span>
          <span className="text-xs font-semibold uppercase tracking-wide text-content-secondary whitespace-nowrap truncate">
            Create KV Store
          </span>
        </Link>
      )}

      <SidebarResourceList
        connectionId={connectionId}
        section="kv"
        names={buckets}
        selectedName={selectedBucket}
        hrefFor={bucketHref}
        noun="KV bucket"
        isRefreshing={isFetching}
        onRefresh={() => refetch()}
      />
    </div>
  )
}
