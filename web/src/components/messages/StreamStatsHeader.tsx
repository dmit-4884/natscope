import { useMemo } from 'react'
import { useStreamDetail } from '@/contexts/streams'
import { useLiveStatsStore } from '@/contexts/live'
import { formatNumber, formatBytes, formatBytesPerSecond } from '@/utils/formatters'

interface StreamStatsHeaderProps {
  streamName: string
  connectionId: string
}

export default function StreamStatsHeader({ streamName, connectionId }: StreamStatsHeaderProps) {
  const { data: streamDetail } = useStreamDetail(streamName, connectionId)
  const liveStats = useLiveStatsStore((s) => s.stats)

  const stats = useMemo(() => {
    if (!streamDetail) return null
    return {
      messages: streamDetail.state?.messages || streamDetail.messages || 0,
      consumers: streamDetail.consumer_count || streamDetail.state?.consumer_count || 0,
      bytes: streamDetail.state?.bytes || streamDetail.bytes || 0,
      cluster: streamDetail.cluster?.name || null,
    }
  }, [streamDetail])

  if (!stats) {
    return (
      <div className="px-4 py-2 bg-surface-primary border-b flex items-center gap-6 text-sm animate-pulse">
        <div className="h-4 w-24 bg-surface-hover rounded" />
        <div className="h-4 w-20 bg-surface-hover rounded" />
        <div className="h-4 w-24 bg-surface-hover rounded" />
      </div>
    )
  }

  return (
    <div className="px-4 py-2 bg-surface-primary border-b flex items-center gap-x-6 gap-y-1 text-sm flex-wrap">
      {/* Messages */}
      <div className="flex items-center gap-1.5">
        <span className="text-content-tertiary">Messages</span>
        <span className="font-semibold text-content-primary">{formatNumber(stats.messages)}</span>
      </div>

      {/* Consumers */}
      <div className="flex items-center gap-1.5">
        <span className="text-content-tertiary">Consumers</span>
        <span className="font-semibold text-content-primary">{stats.consumers}</span>
      </div>

      {/* Size */}
      <div className="flex items-center gap-1.5">
        <span className="text-content-tertiary">Size</span>
        <span className="font-semibold text-content-primary">{formatBytes(stats.bytes)}</span>
      </div>

      {/* Messages/s */}
      <div className="flex items-center gap-1.5">
        <span className="text-content-tertiary">Messages/s</span>
        <span className={`font-semibold ${liveStats?.isConnected ? 'text-content-primary' : 'text-content-tertiary'}`}>
          {liveStats?.msgPerSecond?.toFixed(0) || '0'}
        </span>
      </div>

      {/* Bytes/s — estimated from msg/s × avg msg size */}
      <div className="flex items-center gap-1.5">
        <span className="text-content-tertiary">Bytes/s</span>
        <span className={`font-semibold ${liveStats?.isConnected ? 'text-content-primary' : 'text-content-tertiary'}`}>
          {liveStats?.isConnected && stats.messages > 0
            ? formatBytesPerSecond((stats.bytes / stats.messages) * (liveStats?.msgPerSecond || 0))
            : '0 B/s'}
        </span>
      </div>

      {/* Cluster */}
      {stats.cluster && (
        <div className="flex items-center gap-1.5">
          <span className="text-content-tertiary">Cluster</span>
          <span className="font-semibold text-content-primary">{stats.cluster}</span>
        </div>
      )}

    </div>
  )
}
