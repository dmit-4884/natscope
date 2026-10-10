import { useLayoutEffect, useMemo, useRef, useState } from 'react'
import { flushSync } from 'react-dom'
import Tooltip from '@/components/common/Tooltip'
import { useStreamDetail } from '@/contexts/streams'
import { useLiveStatsStore } from '@/contexts/live'
import { cn } from '@/utils/cn'
import { formatNumber, formatBytes, formatBytesPerSecond } from '@/utils/formatters'

interface StreamStatsHeaderProps {
  streamName: string
  connectionId: string
}

interface Stat {
  key: string
  label: string
  short: string
  value: string
  muted?: boolean
  optional?: boolean
  hint?: string
}

const RATE_OFF = '—'
const RATE_HINT = 'The rate is measured only while Realtime is on'
const FULL = 0
const COMPACT = 1
const MINIMAL = 2
const LEVELS = [FULL, COMPACT, MINIMAL]

function StatsRow({ stats, level }: { stats: Stat[]; level: number }) {
  return (
    <>
      {stats
        .filter((s) => level < MINIMAL || !s.optional)
        .map((s) => (
          <div key={s.key} className="flex items-center gap-1.5 whitespace-nowrap">
            {level === FULL || s.short === s.label ? (
              <span className="text-content-tertiary">{s.label}</span>
            ) : (
              <abbr title={s.label} className="text-content-tertiary no-underline cursor-help">
                {s.short}
              </abbr>
            )}
            <Tooltip content={s.hint ?? ''}>
              <span className={cn('font-semibold', s.muted ? 'text-content-tertiary' : 'text-content-primary')}>
                {s.value}
              </span>
            </Tooltip>
          </div>
        ))}
    </>
  )
}

export default function StreamStatsHeader({ streamName, connectionId }: StreamStatsHeaderProps) {
  const { data: streamDetail } = useStreamDetail(streamName, connectionId)
  const liveStats = useLiveStatsStore((s) => s.stats)
  const containerRef = useRef<HTMLDivElement>(null)
  const measureRefs = useRef<(HTMLDivElement | null)[]>([])
  const [level, setLevel] = useState(FULL)

  const stats = useMemo((): Stat[] | null => {
    if (!streamDetail) return null
    const messages = streamDetail.state?.messages || streamDetail.messages || 0
    const bytes = streamDetail.state?.bytes || streamDetail.bytes || 0
    const cluster = streamDetail.cluster?.name || null
    const live = Boolean(liveStats?.isConnected)
    const items: Stat[] = [
      { key: 'messages', label: 'Messages', short: 'Msgs', value: formatNumber(messages) },
      {
        key: 'consumers',
        label: 'Consumers',
        short: 'Cons',
        value: String(streamDetail.consumer_count || streamDetail.state?.consumer_count || 0),
      },
      { key: 'size', label: 'Size', short: 'Size', value: formatBytes(bytes) },
      {
        key: 'rate',
        label: 'Messages/s',
        short: 'Msg/s',
        value: live ? liveStats?.msgPerSecond?.toFixed(0) || '0' : RATE_OFF,
        muted: !live,
        hint: live ? undefined : RATE_HINT,
      },
      {
        key: 'throughput',
        label: 'Bytes/s',
        short: 'B/s',
        value:
          live ? (messages > 0 ? formatBytesPerSecond((bytes / messages) * (liveStats?.msgPerSecond || 0)) : '0 B/s') : RATE_OFF,
        muted: !live,
        hint: live ? undefined : RATE_HINT,
        optional: true,
      },
    ]
    if (cluster) items.push({ key: 'cluster', label: 'Cluster', short: 'Cluster', value: cluster, optional: true })
    return items
  }, [streamDetail, liveStats])

  useLayoutEffect(() => {
    const container = containerRef.current
    if (!container || !stats) return
    const fit = () => {
      if (container.clientWidth === 0) return
      const style = getComputedStyle(container)
      const available =
        container.clientWidth - (parseFloat(style.paddingLeft) || 0) - (parseFloat(style.paddingRight) || 0)
      const fitting = LEVELS.find((l) => (measureRefs.current[l]?.offsetWidth ?? Infinity) <= available)
      setLevel(fitting ?? LEVELS.length)
    }
    fit()
    if (typeof ResizeObserver === 'undefined') return
    const observer = new ResizeObserver(() => flushSync(fit))
    observer.observe(container)
    return () => observer.disconnect()
  }, [stats])

  if (!stats) {
    return (
      <div className="px-4 py-2 bg-surface-primary border-b flex items-center gap-6 text-sm reveal-after-delay">
        <div className="h-5 w-24 bg-surface-hover rounded animate-pulse" />
        <div className="h-5 w-20 bg-surface-hover rounded animate-pulse" />
        <div className="h-5 w-24 bg-surface-hover rounded animate-pulse" />
      </div>
    )
  }

  return (
    <div
      ref={containerRef}
      data-testid="stream-stats"
      className={cn(
        'relative px-4 py-2 bg-surface-primary border-b flex items-center gap-y-1 text-sm',
        level === FULL ? 'gap-x-6' : 'gap-x-4',
        level >= LEVELS.length ? 'flex-wrap' : 'overflow-hidden',
      )}
    >
      <StatsRow stats={stats} level={Math.min(level, MINIMAL)} />
      <div aria-hidden="true" className="invisible absolute left-0 top-0 h-0 w-0 overflow-hidden">
        {LEVELS.map((l) => (
          <div
            key={l}
            ref={(el) => {
              measureRefs.current[l] = el
            }}
            data-level={l}
            className={cn('flex w-max items-center', l === FULL ? 'gap-x-6' : 'gap-x-4')}
          >
            <StatsRow stats={stats} level={l} />
          </div>
        ))}
      </div>
    </div>
  )
}
