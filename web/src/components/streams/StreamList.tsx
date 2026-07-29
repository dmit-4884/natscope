import { useMemo } from 'react'
import { Link, useParams } from 'react-router-dom'
import { Code } from '@connectrpc/connect'
import { useStreamEntities, filterRegularStreams, type Stream } from '@/contexts/streams'
import ErrorAlert from '@/components/ui/ErrorAlert'
import { EmptyState, PlusIcon, RefreshIcon, SkeletonRows } from '@/components/ui'
import Tooltip from '@/components/common/Tooltip'
import { getErrorMessage, isErrorCode } from '@/api/errors'

const STREAMS_ICON = (
  <svg fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
  </svg>
)

interface StreamListProps {
  connectionId: string
}

function StreamItem({
  stream,
  isSelected,
}: {
  stream: Stream
  isSelected: boolean
}) {
  const streamName = stream.name.value

  return (
    <Link
      to={`/streams/${encodeURIComponent(streamName)}`}
      className={`group block px-3 py-2 transition-all ${
        isSelected
          ? 'bg-accent-light border-l-2 border-l-blue-500'
          : 'hover:bg-surface-secondary border-l-2 border-l-transparent'
      }`}
      title={streamName}
    >
      <span className={`text-sm truncate block ${
        isSelected ? 'font-medium text-content-primary' : 'text-gray-700'
      }`}>
        {streamName}
      </span>
    </Link>
  )
}

export default function StreamList({ connectionId }: StreamListProps) {
  const { streamName: selectedStream } = useParams()
  const { streams: allStreams, isLoading, error, refetch, isFetching } = useStreamEntities(connectionId)

  // Filter to only show regular streams (KV and Object stores are in separate sections)
  const streams = useMemo(
    () => filterRegularStreams(allStreams),
    [allStreams]
  )

  if (isLoading) {
    return <SkeletonRows count={6} rowClassName="h-9" className="p-2" />
  }

  if (error) {
    const isConnectionError = isErrorCode(error, Code.Unavailable)
    return (
      <div className="p-4">
        <ErrorAlert message={isConnectionError ? `NATS server unavailable` : `Failed to load streams: ${getErrorMessage(error)}`} />
      </div>
    )
  }

  if (streams.length === 0) {
    return (
      <EmptyState
        size="sm"
        icon={STREAMS_ICON}
        title="No streams"
        description="Create your first stream to start publishing messages."
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
              to={`/streams/new`}
              className="inline-flex items-center gap-1.5 text-sm text-accent hover:text-accent-text"
            >
              <PlusIcon className="w-4 h-4" />
              Create stream
            </Link>
          </div>
        }
      />
    )
  }

  return (
    <div>
      <div className="flex justify-end border-b border-border bg-surface-secondary">
        <Tooltip content="Refresh streams">
          <button
            onClick={() => refetch()}
            disabled={isFetching}
            className="px-2.5 py-1.5 text-content-muted hover:text-content-secondary hover:bg-surface-tertiary transition-colors disabled:opacity-50"
            aria-label="Refresh streams"
          >
            <RefreshIcon className={`w-3.5 h-3.5 ${isFetching ? 'animate-spin' : ''}`} />
          </button>
        </Tooltip>
      </div>

      {/* Stream list */}
      {streams.map((stream) => (
        <StreamItem
          key={stream.name.value}
          stream={stream}
          isSelected={stream.name.value === selectedStream}
        />
      ))}
    </div>
  )
}
