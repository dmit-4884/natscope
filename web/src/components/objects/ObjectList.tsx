import { useMemo } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useObjectBuckets } from '@/contexts/objects'
import ErrorAlert from '@/components/ui/ErrorAlert'
import { EmptyState, PlusIcon, RefreshIcon, SkeletonRows } from '@/components/ui'
import Tooltip from '@/components/common/Tooltip'
import { getErrorMessage } from '@/api/errors'
import type { ObjectBucketInfo } from '@/types/management'

const OBJECTS_ICON = (
  <svg fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4" />
  </svg>
)

interface ObjectListProps {
  connectionId: string
}

function ObjectItem({
  bucket,
  isSelected,
}: {
  bucket: ObjectBucketInfo
  isSelected: boolean
}) {
  return (
    <Link
      to={`/objects/${encodeURIComponent(bucket.bucket)}`}
      className={`group block px-3 py-2 transition-all ${
        isSelected
          ? 'bg-accent-light border-l-2 border-l-blue-500'
          : 'hover:bg-surface-secondary border-l-2 border-l-transparent'
      }`}
      title={bucket.bucket}
    >
      <div className="flex items-center gap-2 min-w-0">
        <span className={`text-sm truncate ${
          isSelected ? 'font-medium text-content-primary' : 'text-gray-700'
        }`}>
          {bucket.bucket}
        </span>
        {bucket.sealed && (
          <svg className="w-3 h-3 text-orange-500 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
          </svg>
        )}
      </div>
    </Link>
  )
}

export default function ObjectList({ connectionId }: ObjectListProps) {
  const { bucketName: selectedBucket } = useParams()
  const { data: buckets, isLoading, isFetching, error, refetch } = useObjectBuckets(connectionId)

  const sortedBuckets = useMemo(() => {
    if (!buckets) return []
    return [...buckets].sort((a, b) => a.bucket.localeCompare(b.bucket))
  }, [buckets])

  if (isLoading) {
    return <SkeletonRows count={5} rowClassName="h-9" className="p-2" />
  }

  if (error) {
    return (
      <div className="p-4">
        <ErrorAlert message={`Failed to load object stores: ${getErrorMessage(error)}`} />
      </div>
    )
  }

  if (sortedBuckets.length === 0) {
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
            <Link
              to={`/objects/new`}
              className="inline-flex items-center gap-1.5 text-sm text-accent hover:text-accent-text"
            >
              <PlusIcon className="w-4 h-4" />
              Create object store
            </Link>
          </div>
        }
      />
    )
  }

  return (
    <div>
      {/* Toolbar */}
      <div className="flex items-center border-b border-border bg-surface-secondary">
        <Link
          to={`/objects/new`}
          className="flex-1 px-3 py-2.5 flex items-center gap-2 hover:bg-surface-tertiary transition-colors"
        >
          <span className="text-content-muted">
            <PlusIcon className="w-4 h-4" />
          </span>
          <span className="text-xs font-semibold uppercase tracking-wide text-content-secondary whitespace-nowrap truncate">
            Create Object Store
          </span>
        </Link>
        <Tooltip content="Refresh object stores">
          <button
            onClick={() => refetch()}
            disabled={isFetching}
            className="shrink-0 px-2.5 py-2.5 text-content-muted hover:text-content-secondary hover:bg-surface-tertiary transition-colors disabled:opacity-50"
            aria-label="Refresh object stores"
          >
            <RefreshIcon className={`w-3.5 h-3.5 ${isFetching ? 'animate-spin' : ''}`} />
          </button>
        </Tooltip>
      </div>

      {/* Bucket list */}
      {sortedBuckets.map((bucket) => (
        <ObjectItem
          key={bucket.bucket}
          bucket={bucket}
          isSelected={bucket.bucket === selectedBucket}
        />
      ))}
    </div>
  )
}
