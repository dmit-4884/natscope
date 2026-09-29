import { useEffect, useLayoutEffect, useRef, type PointerEvent, type ReactNode } from 'react'
import type { StreamRelationEdge, StreamRelationNode } from '@/types/nats'
import { CloseIcon, DragHandleIcon } from '@/components/ui'
import { cn } from '@/utils/cn'
import { formatCount, formatDateTime } from '@/utils/formatters'
import Tooltip from '../../common/Tooltip'
import { RELATION_STYLES, externalLabel, lastSeen } from './relationStyles'

function Row({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="grid grid-cols-[6.5rem_1fr] gap-2 py-1.5">
      <dt className="text-xs text-content-tertiary">{label}</dt>
      <dd className="min-w-0 break-words text-xs text-content-primary">{children}</dd>
    </div>
  )
}

function Filters({ edge }: { edge: StreamRelationEdge }) {
  const transforms = edge.source?.subject_transforms
  if (transforms?.length) {
    return (
      <ul className="space-y-0.5">
        {transforms.map((t) => (
          <li key={`${t.src}\n${t.dest}`}>
            <code>{t.src}</code>
            {t.dest ? <> → <code>{t.dest}</code></> : <span className="text-content-tertiary"> as is</span>}
          </li>
        ))}
      </ul>
    )
  }
  if (edge.source?.filter_subject) return <code>{edge.source.filter_subject}</code>
  return <span className="text-content-tertiary">All subjects</span>
}

export interface PanelPosition {
  x: number
  y: number
}

interface RelationDetailsProps {
  edge: StreamRelationEdge
  nodes: ReadonlyMap<string, StreamRelationNode>
  position: PanelPosition | null
  onMove: (position: PanelPosition | null) => void
  onClose: () => void
}

export function RelationDetails({ edge, nodes, position, onMove, onClose }: RelationDetailsProps) {
  const panelRef = useRef<HTMLElement>(null)
  const drag = useRef<{ dx: number; dy: number } | null>(null)
  const style = RELATION_STYLES[edge.kind]
  const from = nodes.get(edge.from)
  const to = nodes.get(edge.to)
  const source = edge.source
  const state = edge.state
  const mirrorDirect = edge.kind === 'mirror' && to?.info?.config.mirror_direct

  const clamp = (x: number, y: number): PanelPosition => {
    const panel = panelRef.current
    const parent = panel?.offsetParent
    if (!panel || !(parent instanceof HTMLElement)) return { x, y }
    return {
      x: Math.max(0, Math.min(x, parent.clientWidth - panel.offsetWidth)),
      y: Math.max(0, Math.min(y, parent.clientHeight - panel.offsetHeight)),
    }
  }

  const refit = () => {
    if (!position) return
    const fitted = clamp(position.x, position.y)
    if (fitted.x !== position.x || fitted.y !== position.y) onMove(fitted)
  }

  useLayoutEffect(refit)

  useEffect(() => {
    window.addEventListener('resize', refit)
    return () => window.removeEventListener('resize', refit)
  })

  const handlePointerDown = (e: PointerEvent<HTMLElement>) => {
    const panel = panelRef.current
    if (e.button !== 0 || !panel || (e.target as Element).closest('button')) return
    const rect = panel.getBoundingClientRect()
    drag.current = { dx: e.clientX - rect.left, dy: e.clientY - rect.top }
    e.currentTarget.setPointerCapture?.(e.pointerId)
    e.preventDefault()
  }

  const handlePointerMove = (e: PointerEvent<HTMLElement>) => {
    const grab = drag.current
    const parent = panelRef.current?.offsetParent
    if (!grab || !(parent instanceof HTMLElement)) return
    const origin = parent.getBoundingClientRect()
    onMove(clamp(e.clientX - origin.left - grab.dx, e.clientY - origin.top - grab.dy))
  }

  return (
    <section
      ref={panelRef}
      aria-label={`${style.label} link details`}
      className={cn(
        'absolute z-dropdown w-80 max-h-[calc(100%-1.5rem)] overflow-auto rounded-lg border border-border bg-surface-primary shadow-lg',
        !position && 'right-3 top-3',
      )}
      style={position ? { left: position.x, top: position.y } : undefined}
      data-testid="relation-details"
      onKeyDown={(e) => {
        if (e.key === 'Escape') onClose()
      }}
    >
      <header
        className="flex cursor-move select-none touch-none items-center gap-2 border-b border-border px-3 py-2"
        onPointerDown={handlePointerDown}
        onPointerMove={handlePointerMove}
        onPointerUp={() => {
          drag.current = null
        }}
        onPointerCancel={() => {
          drag.current = null
        }}
        onDoubleClick={() => onMove(null)}
        data-testid="relation-details-handle"
      >
        <Tooltip content="Drag to move, double-click to dock back">
          <span className="shrink-0 text-content-muted">
            <DragHandleIcon className="w-3 h-3" />
          </span>
        </Tooltip>
        <span className={cn('h-0.5 w-4 rounded-full', style.swatch)} aria-hidden="true" />
        <h3 className="text-sm font-semibold text-content-primary">{style.label}</h3>
        <button
          type="button"
          onClick={onClose}
          aria-label="Close link details"
          className="ml-auto rounded p-1 text-content-muted hover:bg-surface-tertiary hover:text-content-secondary"
        >
          <CloseIcon className="w-3.5 h-3.5" />
        </button>
      </header>

      <dl className="divide-y divide-border px-3 py-1">
        <Row label="From">{from?.name ?? edge.from}</Row>
        <Row label="To">{to?.name ?? edge.to}</Row>

        {source && (
          <>
            <Row label="Subjects">
              <Filters edge={edge} />
            </Row>
            {source.opt_start_seq !== undefined && <Row label="Starts at">sequence {formatCount(source.opt_start_seq)}</Row>}
            {source.opt_start_time !== undefined && <Row label="Starts at">{formatDateTime(source.opt_start_time)}</Row>}
            {source.external && (
              <Row label="External">
                {externalLabel(source.external)}
                {source.external.deliver_prefix && (
                  <span className="block text-content-tertiary">deliver <code>{source.external.deliver_prefix}</code></span>
                )}
              </Row>
            )}
          </>
        )}

        {mirrorDirect && <Row label="Direct get">Served by the mirror too</Row>}

        {state && (
          <>
            <Row label="Lag">
              {state.active_ns < 0 ? 'Unknown' : state.lag === 0 ? 'Caught up' : `${formatCount(state.lag)} messages behind`}
            </Row>
            <Row label="Last seen">
              <span className={cn(state.active_ns < 0 && 'text-status-warning-text')}>{lastSeen(state.active_ns)}</span>
            </Row>
            {state.error && (
              <Row label="Error">
                <span className="text-status-error-text">{state.error}</span>
              </Row>
            )}
          </>
        )}
        {source && !state && <Row label="Status"><span className="text-content-tertiary">Not reported by the server</span></Row>}

        {edge.republish && (
          <>
            <Row label="Republishes"><code>{edge.republish.src || '>'}</code></Row>
            <Row label="To subject"><code>{edge.republish.dest}</code></Row>
            <Row label="Payload">{edge.republish.headers_only ? 'Headers only' : 'Headers and body'}</Row>
          </>
        )}
      </dl>
    </section>
  )
}
