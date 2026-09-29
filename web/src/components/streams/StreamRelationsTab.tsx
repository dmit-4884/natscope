import { useEffect, useMemo, useState } from 'react'
import { Link, useOutletContext } from 'react-router-dom'
import { useStreamRelations } from '@/contexts/streams'
import type { StreamRelationKind } from '@/types/nats'
import { EmptyState, MinusIcon, PlusIcon, QueryErrorState, RefreshIcon, RelationsIcon, SkeletonRows } from '@/components/ui'
import { cn } from '@/utils/cn'
import { parseIntOr } from '@/utils/numbers'
import { plural } from '@/utils/plural'
import { safeGetItem, safeSetItem } from '@/utils/safeStorage'
import Tooltip from '../common/Tooltip'
import type { StreamViewOutletContext } from './StreamView'
import { layoutGraph, neighborhood } from './relations/graph'
import { RelationsCanvas } from './relations/RelationsCanvas'

const DEPTH_KEY = 'nats_relations_depth'
const DEFAULT_DEPTH = 3
const MIN_DEPTH = 1
const MAX_DEPTH = 10

const clampDepth = (depth: number) => Math.min(MAX_DEPTH, Math.max(MIN_DEPTH, depth))

export default function StreamRelationsTab() {
  const { connectionId, streamName } = useOutletContext<StreamViewOutletContext>()
  const { data, error, isLoading, isFetching, refetch } = useStreamRelations(connectionId)
  const [depth, setDepth] = useState(() => clampDepth(parseIntOr(safeGetItem(DEPTH_KEY) ?? '', DEFAULT_DEPTH)))
  const [hidden, setHidden] = useState<ReadonlySet<StreamRelationKind>>(() => new Set())

  const visible = useMemo(() => (data ? neighborhood(data, streamName, depth, hidden) : null), [data, streamName, depth, hidden])
  const layout = useMemo(() => (visible && visible.nodes.length > 0 ? layoutGraph(visible) : null), [visible])
  const fitKey = `${streamName}\n${depth}\n${[...hidden].sort().join()}`

  useEffect(() => {
    safeSetItem(DEPTH_KEY, String(depth))
  }, [depth])

  const changeDepth = (delta: number) => setDepth((prev) => clampDepth(prev + delta))

  const toggleKind = (kind: StreamRelationKind) => {
    setHidden((prev) => {
      const next = new Set(prev)
      if (!next.delete(kind)) next.add(kind)
      return next
    })
  }

  if (isLoading) {
    return (
      <div className="flex-1 p-4">
        <SkeletonRows count={4} rowClassName="h-10" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex-1 p-4">
        <QueryErrorState error={error} onRetry={() => refetch()} title="Could not load stream relations" />
      </div>
    )
  }

  if (!layout || !visible) {
    return (
      <div className="flex-1 flex items-center justify-center p-4">
        <EmptyState
          icon={<RelationsIcon className="w-10 h-10" />}
          title="No relations"
          description="This stream has no sources or mirror and does not republish, and no other stream sources it, mirrors it or republishes into it."
          action={
            <Link to="../config" relative="path" className="text-sm text-accent hover:text-accent-hover">
              Open config
            </Link>
          }
        />
      </div>
    )
  }

  const streamCount = visible.nodes.length
  const linkCount = visible.edges.length

  return (
    <div className="flex-1 flex flex-col min-h-0">
      <div className="flex items-center gap-3 border-b border-border bg-surface-primary px-4 py-2">
        <p className="text-xs text-content-tertiary" data-testid="relations-summary">
          {plural(streamCount, 'node')} · {plural(linkCount, 'link')}
        </p>
        <div className="ml-auto flex items-center gap-2">
          <div role="group" aria-label="Depth" className="flex items-center gap-1">
            <span className="text-xs text-content-secondary">Depth</span>
            <div className="flex items-center rounded border border-border">
              <button
                type="button"
                onClick={() => changeDepth(-1)}
                disabled={depth <= MIN_DEPTH}
                aria-label="Decrease depth"
                className="p-1 text-content-secondary hover:bg-surface-tertiary disabled:opacity-40"
              >
                <MinusIcon className="w-3.5 h-3.5" />
              </button>
              <output className="w-6 text-center text-xs font-medium tabular-nums text-content-primary" aria-live="polite">
                {depth}
              </output>
              <button
                type="button"
                onClick={() => changeDepth(1)}
                disabled={depth >= MAX_DEPTH}
                aria-label="Increase depth"
                className="p-1 text-content-secondary hover:bg-surface-tertiary disabled:opacity-40"
              >
                <PlusIcon className="w-3.5 h-3.5" />
              </button>
            </div>
          </div>
          <Tooltip content="Refresh relations">
            <button
              type="button"
              onClick={() => refetch()}
              disabled={isFetching}
              aria-label="Refresh relations"
              className="p-1.5 rounded text-content-muted hover:text-content-secondary hover:bg-surface-tertiary transition-colors disabled:opacity-50"
            >
              <RefreshIcon className={cn('w-3.5 h-3.5', isFetching && 'animate-spin')} />
            </button>
          </Tooltip>
        </div>
      </div>

      <RelationsCanvas layout={layout} rootId={streamName} fitKey={fitKey} hidden={hidden} onToggleKind={toggleKind} />
    </div>
  )
}
