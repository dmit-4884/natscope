import { useCallback, useEffect, useId, useLayoutEffect, useMemo, useRef, useState, type KeyboardEvent, type PointerEvent } from 'react'
import type { StreamRelationKind, StreamRelationNode } from '@/types/nats'
import { FitViewIcon, MinusIcon, PlusIcon, WarningIcon } from '@/components/ui'
import { cn } from '@/utils/cn'
import { safeGetItem, safeRemoveItem, safeSetItem } from '@/utils/safeStorage'
import Tooltip from '../../common/Tooltip'
import type { EdgePath, GraphLayout } from './graph'
import { RelationDetails, type PanelPosition } from './RelationDetails'
import { RelationNodeCard } from './RelationNodeCard'
import { RELATION_KINDS, RELATION_STYLES, linkChip, linkProblem } from './relationStyles'

const PADDING = 48
const MIN_ZOOM = 0.2
const MAX_ZOOM = 2
const ZOOM_STEP = 1.25
const PAN_STEP = 60
const GRID = 20
const DETAILS_POSITION_KEY = 'nats_relations_details_position'

function storedDetailsPosition(): PanelPosition | null {
  try {
    const value = JSON.parse(safeGetItem(DETAILS_POSITION_KEY) ?? 'null') as unknown
    if (value && typeof value === 'object' && 'x' in value && 'y' in value) {
      const { x, y } = value as { x: unknown; y: unknown }
      if (typeof x === 'number' && typeof y === 'number' && Number.isFinite(x) && Number.isFinite(y)) return { x, y }
    }
  } catch {
    return null
  }
  return null
}

interface View {
  x: number
  y: number
  k: number
}

const clampZoom = (k: number) => Math.min(MAX_ZOOM, Math.max(MIN_ZOOM, k))

function zoomAt(view: View, k: number, px: number, py: number): View {
  const next = clampZoom(k)
  return { k: next, x: px - ((px - view.x) * next) / view.k, y: py - ((py - view.y) * next) / view.k }
}

interface RelationsCanvasProps {
  layout: GraphLayout
  rootId: string
  fitKey: string
  hidden: ReadonlySet<StreamRelationKind>
  onToggleKind: (kind: StreamRelationKind) => void
}

export function RelationsCanvas({ layout, rootId, fitKey, hidden, onToggleKind }: RelationsCanvasProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const layoutRef = useRef(layout)
  const drag = useRef<{ x: number; y: number } | null>(null)
  const userMoved = useRef(false)
  const lastSize = useRef<{ width: number; height: number } | null>(null)
  const [view, setView] = useState<View>({ x: PADDING, y: PADDING, k: 1 })
  const [selectedKey, setSelectedKey] = useState<string | null>(null)
  const [detailsPosition, setDetailsPosition] = useState<PanelPosition | null>(storedDetailsPosition)
  const markerId = useId()

  const nodesById = useMemo(
    () => new Map<string, StreamRelationNode>(layout.nodes.map((b) => [b.node.id, b.node])),
    [layout],
  )
  const selected = layout.edges.find((e) => e.key === selectedKey)

  const moveView = useCallback((update: (v: View) => View) => {
    userMoved.current = true
    setView(update)
  }, [])

  const fit = useCallback(() => {
    userMoved.current = false
    const el = containerRef.current
    if (!el) return
    const { width, height } = el.getBoundingClientRect()
    const content = layoutRef.current
    if (width === 0 || height === 0) {
      setView({ x: PADDING, y: PADDING, k: 1 })
      return
    }
    const k = clampZoom(Math.min(1, (width - 2 * PADDING) / content.width, (height - 2 * PADDING) / content.height))
    setView({ k, x: (width - content.width * k) / 2, y: (height - content.height * k) / 2 })
  }, [])

  useLayoutEffect(() => {
    layoutRef.current = layout
  })

  useLayoutEffect(() => {
    fit()
    setSelectedKey(null)
  }, [fitKey, fit])

  useLayoutEffect(() => {
    if (!userMoved.current) fit()
  }, [layout.width, layout.height, fit])

  useEffect(() => {
    if (detailsPosition) safeSetItem(DETAILS_POSITION_KEY, JSON.stringify(detailsPosition))
    else safeRemoveItem(DETAILS_POSITION_KEY)
  }, [detailsPosition])

  useEffect(() => {
    const el = containerRef.current
    if (!el) return
    const onWheel = (e: WheelEvent) => {
      if ((e.target as Element).closest('section')) return
      e.preventDefault()
      const rect = el.getBoundingClientRect()
      const factor = Math.exp(-e.deltaY * (e.ctrlKey ? 0.01 : 0.002))
      moveView((v) => zoomAt(v, v.k * factor, e.clientX - rect.left, e.clientY - rect.top))
    }
    el.addEventListener('wheel', onWheel, { passive: false })
    return () => el.removeEventListener('wheel', onWheel)
  }, [moveView])

  useEffect(() => {
    const el = containerRef.current
    if (!el || typeof ResizeObserver === 'undefined') return
    const observer = new ResizeObserver(([entry]) => {
      const { width, height } = entry.contentRect
      const previous = lastSize.current
      lastSize.current = { width, height }
      if (!previous || (previous.width === width && previous.height === height)) return
      if (!userMoved.current) {
        fit()
        return
      }
      setView((v) => ({ ...v, x: v.x + (width - previous.width) / 2, y: v.y + (height - previous.height) / 2 }))
    })
    observer.observe(el)
    return () => observer.disconnect()
  }, [fit])

  const zoomBy = (factor: number) => {
    const rect = containerRef.current?.getBoundingClientRect()
    moveView((v) => zoomAt(v, v.k * factor, (rect?.width ?? 0) / 2, (rect?.height ?? 0) / 2))
  }

  const handlePointerDown = (e: PointerEvent<HTMLDivElement>) => {
    if (e.button !== 0 || (e.target as Element).closest('a, button, section')) return
    drag.current = { x: e.clientX, y: e.clientY }
    e.currentTarget.setPointerCapture?.(e.pointerId)
  }

  const handlePointerMove = (e: PointerEvent<HTMLDivElement>) => {
    const start = drag.current
    if (!start) return
    drag.current = { x: e.clientX, y: e.clientY }
    moveView((v) => ({ ...v, x: v.x + e.clientX - start.x, y: v.y + e.clientY - start.y }))
  }

  const endDrag = () => {
    drag.current = null
  }

  const handleKeyDown = (e: KeyboardEvent<HTMLDivElement>) => {
    if (e.key === 'Escape' && selectedKey !== null) {
      setSelectedKey(null)
      e.preventDefault()
      return
    }
    if (e.target !== e.currentTarget) return
    const pan: Record<string, [number, number]> = {
      ArrowLeft: [PAN_STEP, 0],
      ArrowRight: [-PAN_STEP, 0],
      ArrowUp: [0, PAN_STEP],
      ArrowDown: [0, -PAN_STEP],
    }
    if (pan[e.key]) {
      const [dx, dy] = pan[e.key]
      moveView((v) => ({ ...v, x: v.x + dx, y: v.y + dy }))
    } else if (e.key === '+' || e.key === '=') zoomBy(ZOOM_STEP)
    else if (e.key === '-') zoomBy(1 / ZOOM_STEP)
    else if (e.key === '0') fit()
    else return
    e.preventDefault()
  }

  const describe = (edge: EdgePath) =>
    `${RELATION_STYLES[edge.edge.kind].label} from ${nodesById.get(edge.edge.from)?.name ?? edge.edge.from} to ${
      nodesById.get(edge.edge.to)?.name ?? edge.edge.to
    }`

  return (
    <div
      ref={containerRef}
      role="group"
      aria-label="Stream relations graph. Drag or use the arrow keys to pan, plus and minus to zoom, 0 to fit."
      tabIndex={0}
      className="canvas-dots relative flex-1 min-h-0 overflow-hidden bg-surface-secondary cursor-grab active:cursor-grabbing select-none touch-none"
      style={{ backgroundSize: `${GRID * view.k}px ${GRID * view.k}px`, backgroundPosition: `${view.x}px ${view.y}px` }}
      onPointerDown={handlePointerDown}
      onPointerMove={handlePointerMove}
      onPointerUp={endDrag}
      onPointerCancel={endDrag}
      onKeyDown={handleKeyDown}
      data-testid="relations-canvas"
    >
      <div
        className="absolute left-0 top-0 origin-top-left"
        style={{ transform: `translate(${view.x}px, ${view.y}px) scale(${view.k})`, width: layout.width, height: layout.height }}
      >
        <svg className="absolute left-0 top-0 overflow-visible" width={layout.width} height={layout.height} aria-hidden="true">
          <defs>
            {RELATION_KINDS.map((kind) => (
              <marker
                key={kind}
                id={`${markerId}-${kind}`}
                viewBox="0 0 10 10"
                refX="9"
                refY="5"
                markerWidth="7"
                markerHeight="7"
                orient="auto-start-reverse"
              >
                <path d="M 0 0 L 10 5 L 0 10 z" className={RELATION_STYLES[kind].fill} />
              </marker>
            ))}
          </defs>
          {layout.edges.map((e) => {
            const problem = linkProblem(e.edge)
            return (
              <path
                key={e.key}
                d={e.path}
                className={cn('fill-none', problem === 'error' ? 'stroke-red-500' : RELATION_STYLES[e.edge.kind].stroke)}
                strokeWidth={e.key === selectedKey ? 3 : 1.75}
                strokeDasharray={problem ? '6 4' : undefined}
                markerEnd={`url(#${markerId}-${e.edge.kind})`}
              />
            )
          })}
        </svg>

        {layout.nodes.map((b) => (
          <RelationNodeCard key={b.node.id} node={b.node} x={b.x} y={b.y} isRoot={b.node.id === rootId} />
        ))}

        {layout.edges.map((e) => {
          const style = RELATION_STYLES[e.edge.kind]
          const { label, details, problem } = linkChip(e.edge)
          const isSelected = e.key === selectedKey
          return (
            <button
              key={e.key}
              type="button"
              onClick={() => setSelectedKey(isSelected ? null : e.key)}
              aria-pressed={isSelected}
              aria-label={`${describe(e)}. Show details`}
              className={cn(
                'absolute -translate-x-1/2 -translate-y-1/2 inline-flex items-center gap-1 whitespace-nowrap rounded-full border px-2 py-0.5 text-2xs font-medium shadow-sm transition-colors',
                problem === 'error' ? 'border-status-error-border bg-status-error-bg text-status-error-text' : style.chip,
                isSelected && 'ring-2 ring-accent ring-offset-1',
              )}
              style={{ left: e.labelX, top: e.labelY }}
              data-testid="relation-edge"
            >
              {problem === 'error' && <WarningIcon className="w-3 h-3" />}
              {label}
              {details.map((detail) => (
                <span key={detail} className="font-normal opacity-80">
                  · {detail}
                </span>
              ))}
            </button>
          )
        })}
      </div>

      <div className="absolute bottom-3 left-3 flex flex-col overflow-hidden rounded-lg border border-border bg-surface-primary shadow-sm">
        <Tooltip content="Zoom in" position="right">
          <button type="button" onClick={() => zoomBy(ZOOM_STEP)} aria-label="Zoom in" className="p-2 text-content-secondary hover:bg-surface-tertiary">
            <PlusIcon />
          </button>
        </Tooltip>
        <Tooltip content="Zoom out" position="right">
          <button type="button" onClick={() => zoomBy(1 / ZOOM_STEP)} aria-label="Zoom out" className="border-y border-border p-2 text-content-secondary hover:bg-surface-tertiary">
            <MinusIcon />
          </button>
        </Tooltip>
        <Tooltip content="Fit to view" position="right">
          <button type="button" onClick={fit} aria-label="Fit to view" className="p-2 text-content-secondary hover:bg-surface-tertiary">
            <FitViewIcon />
          </button>
        </Tooltip>
      </div>

      <div
        role="group"
        aria-label="Link types"
        className="absolute bottom-3 right-3 flex items-center gap-1 rounded-lg border border-border bg-surface-primary px-1.5 py-1 shadow-sm"
      >
        {RELATION_KINDS.map((kind) => {
          const isHidden = hidden.has(kind)
          const style = RELATION_STYLES[kind]
          return (
            <Tooltip key={kind} content={isHidden ? `Show ${kind} links` : `Hide ${kind} links`}>
              <button
                type="button"
                onClick={() => onToggleKind(kind)}
                aria-pressed={!isHidden}
                className={cn(
                  'flex items-center gap-1.5 rounded px-1.5 py-0.5 text-xs hover:bg-surface-tertiary',
                  isHidden ? 'text-content-muted line-through' : 'text-content-secondary',
                )}
              >
                <span className={cn('h-0.5 w-4 rounded-full', style.swatch, isHidden && 'opacity-30')} aria-hidden="true" />
                {style.label}
              </button>
            </Tooltip>
          )
        })}
      </div>

      {selected && (
        <RelationDetails
          edge={selected.edge}
          nodes={nodesById}
          position={detailsPosition}
          onMove={setDetailsPosition}
          onClose={() => setSelectedKey(null)}
        />
      )}
    </div>
  )
}
