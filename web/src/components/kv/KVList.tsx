import { useMemo } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useStreams } from '@/contexts/streams'
import ErrorAlert from '@/components/ui/ErrorAlert'
import { EmptyState, PlusIcon, RefreshIcon, SkeletonRows } from '@/components/ui'
import Tooltip from '@/components/common/Tooltip'
import { getErrorMessage } from '@/api/errors'

const KV_ICON = (
  <svg fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4" />
  </svg>
)

interface KVListProps {
  connectionId: string
}

function KVItem({
  isSelected,
  displayName,
}: {
  isSelected: boolean
  displayName: string
}) {
  return (
    <Link
      to={`/kv/${encodeURIComponent(displayName)}`}
      className={`group block px-3 py-2 transition-all ${
        isSelected
          ? 'bg-accent-light border-l-2 border-l-blue-500'
          : 'hover:bg-surface-secondary border-l-2 border-l-transparent'
      }`}
      title={displayName}
    >
      <span className={`text-sm truncate block ${
        isSelected ? 'font-medium text-content-primary' : 'text-gray-700'
      }`}>
        {displayName}
      </span>
    </Link>
  )
}

export default function KVList({ connectionId }: KVListProps) {
  const { bucketName: selectedBucket } = useParams()
  const { data, isLoading, isFetching, error, refetch } = useStreams(connectionId)

  // Filter to only show KV streams (streams starting with KV_)
  const kvStores = useMemo(() => {
    if (!data?.streams) return []
    return data.streams
      .filter(stream => stream.name.startsWith('KV_'))
      .sort((a, b) => a.name.localeCompare(b.name))
  }, [data?.streams])

  // Get display name (remove KV_ prefix)
  const getDisplayName = (streamName: string) => {
    return streamName.startsWith('KV_') ? streamName.slice(3) : streamName
  }

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

  if (kvStores.length === 0) {
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
            <Link
              to={`/kv/new`}
              className="inline-flex items-center gap-1.5 text-sm text-accent hover:text-accent-text"
            >
              <PlusIcon className="w-4 h-4" />
              Create KV store
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
          to={`/kv/new`}
          className="flex-1 px-3 py-2.5 flex items-center gap-2 hover:bg-surface-tertiary transition-colors"
        >
          <span className="text-content-muted">
            <PlusIcon className="w-4 h-4" />
          </span>
          <span className="text-xs font-semibold uppercase tracking-wide text-content-secondary whitespace-nowrap truncate">
            Create KV Store
          </span>
        </Link>
        <Tooltip content="Refresh KV stores">
          <button
            onClick={() => refetch()}
            disabled={isFetching}
            className="shrink-0 px-2.5 py-2.5 text-content-muted hover:text-content-secondary hover:bg-surface-tertiary transition-colors disabled:opacity-50"
            aria-label="Refresh KV stores"
          >
            <RefreshIcon className={`w-3.5 h-3.5 ${isFetching ? 'animate-spin' : ''}`} />
          </button>
        </Tooltip>
      </div>

      {/* KV list */}
      {kvStores.map((stream) => {
        const displayName = getDisplayName(stream.name)
        return (
          <KVItem
            key={stream.name}
            isSelected={displayName === selectedBucket}
            displayName={displayName}
          />
        )
      })}
    </div>
  )
}
