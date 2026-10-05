import { useMemo } from 'react'
import { Link, useParams } from 'react-router-dom'
import { Code } from '@connectrpc/connect'
import { useConnectionPolicy } from '@/contexts/connection'
import { isRegularStreamName, useStreamNames } from '@/contexts/streams'
import ErrorAlert from '@/components/ui/ErrorAlert'
import { EmptyState, PlusIcon, RefreshIcon, SkeletonRows } from '@/components/ui'
import { SidebarResourceList } from '@/components/common/sidebar/SidebarResourceList'
import { getErrorMessage, isErrorCode } from '@/api/errors'

const STREAMS_ICON = (
  <svg fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
  </svg>
)

interface StreamListProps {
  connectionId: string
}

const streamHref = (name: string) => `/streams/${encodeURIComponent(name)}`

export default function StreamList({ connectionId }: StreamListProps) {
  const { streamName: selectedStream } = useParams()
  const { data: allNames, isLoading, error, refetch, isFetching } = useStreamNames(connectionId)
  const { readOnly } = useConnectionPolicy()

  // Filter to only show regular streams (KV and Object stores are in separate sections)
  const streams = useMemo(() => (allNames ?? []).filter(isRegularStreamName), [allNames])

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
            {!readOnly && (
              <Link
                to={`/streams/new`}
                className="inline-flex items-center gap-1.5 text-sm text-accent hover:text-accent-text"
              >
                <PlusIcon className="w-4 h-4" />
                Create stream
              </Link>
            )}
          </div>
        }
      />
    )
  }

  return (
    <SidebarResourceList
      connectionId={connectionId}
      section="streams"
      names={streams}
      selectedName={selectedStream}
      hrefFor={streamHref}
      noun="stream"
      isRefreshing={isFetching}
      onRefresh={() => refetch()}
    />
  )
}
