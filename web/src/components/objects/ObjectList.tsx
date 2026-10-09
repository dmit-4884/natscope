import { useMemo } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useConnectionPolicy, useSidebarLayoutPending } from '@/contexts/connection'
import { useObjectBuckets } from '@/contexts/objects'
import ErrorAlert from '@/components/ui/ErrorAlert'
import { EmptyState, PlusIcon, RefreshIcon, SkeletonRows } from '@/components/ui'
import { SidebarResourceList } from '@/components/common/sidebar/SidebarResourceList'
import { getErrorMessage } from '@/api/errors'

const OBJECTS_ICON = (
  <svg fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4" />
  </svg>
)

interface ObjectListProps {
  connectionId: string
}

const bucketHref = (name: string) => `/objects/${encodeURIComponent(name)}`

const SEALED_ICON = (
  <svg className="w-3 h-3 text-orange-500 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
  </svg>
)

export default function ObjectList({ connectionId }: ObjectListProps) {
  const { readOnly } = useConnectionPolicy()
  const layoutPending = useSidebarLayoutPending(connectionId)
  const { bucketName: selectedBucket } = useParams()
  const { data: buckets, isLoading, isFetching, error, refetch } = useObjectBuckets(connectionId)

  const bucketNames = useMemo(() => (buckets ?? []).map((b) => b.bucket), [buckets])
  const sealed = useMemo(() => new Set((buckets ?? []).filter((b) => b.sealed).map((b) => b.bucket)), [buckets])

  if (isLoading || layoutPending) {
    return <SkeletonRows count={1} rowClassName="h-9" className="p-2" />
  }

  if (error) {
    return (
      <div className="p-4">
        <ErrorAlert message={`Failed to load object stores: ${getErrorMessage(error)}`} />
      </div>
    )
  }

  if (bucketNames.length === 0) {
    return (
      <EmptyState
        size="sm"
        icon={OBJECTS_ICON}
        title="No object stores"
        description="Create your first bucket to upload and share blobs."
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
                to={`/objects/new`}
                className="inline-flex items-center gap-1.5 text-sm text-accent hover:text-accent-text"
              >
                <PlusIcon className="w-4 h-4" />
                Create object store
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
          to={`/objects/new`}
          className="px-3 py-2.5 flex items-center gap-2 border-b border-border bg-surface-secondary hover:bg-surface-tertiary transition-colors"
        >
          <span className="text-content-muted">
            <PlusIcon className="w-4 h-4" />
          </span>
          <span className="text-xs font-semibold uppercase tracking-wide text-content-secondary whitespace-nowrap truncate">
            Create Object Store
          </span>
        </Link>
      )}

      <SidebarResourceList
        connectionId={connectionId}
        section="objects"
        names={bucketNames}
        selectedName={selectedBucket}
        hrefFor={bucketHref}
        noun="object bucket"
        isRefreshing={isFetching}
        onRefresh={() => refetch()}
        renderBadge={(name) => (sealed.has(name) ? SEALED_ICON : null)}
      />
    </div>
  )
}
