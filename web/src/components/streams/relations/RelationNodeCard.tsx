import { Fragment, type ReactNode } from 'react'
import { Link } from 'react-router-dom'
import type { StreamNodeKind, StreamRelationNode } from '@/types/nats'
import { GlobeIcon, WarningIcon } from '@/components/ui'
import { cn } from '@/utils/cn'
import { formatBytes, formatNumber } from '@/utils/formatters'
import { plural } from '@/utils/plural'
import { NODE_HEIGHT, NODE_WIDTH } from './graph'
import { externalLabel } from './relationStyles'

const KIND_TAG: Partial<Record<StreamNodeKind, string>> = { kv: 'KV bucket', object_store: 'Object store' }

const RETENTION_LABEL: Record<string, string> = { limits: 'Limits', interest: 'Interest', workqueue: 'Work queue' }

function Stat({ children }: { children: ReactNode }) {
  return (
    <span className="rounded bg-surface-tertiary px-1.5 py-px text-2xs text-content-secondary whitespace-nowrap">
      {children}
    </span>
  )
}

function streamStats(node: StreamRelationNode): string[] {
  const info = node.info
  if (!info) return []
  const stats = [
    `${formatNumber(info.messages)} msgs`,
    formatBytes(info.bytes, 1),
    plural(info.subjects.length, 'subject'),
    `R${info.config.num_replicas ?? 1}`,
    info.config.storage === 'memory' ? 'Memory' : 'File',
    RETENTION_LABEL[info.config.retention] ?? info.config.retention,
    plural(info.consumer_count, 'consumer'),
  ]
  if (info.config.compression === 's2') stats.push('S2')
  if (info.config.sealed) stats.push('Sealed')
  if (info.cluster?.name) stats.push(info.cluster.name)
  return stats
}

interface RelationNodeCardProps {
  node: StreamRelationNode
  x: number
  y: number
  isRoot: boolean
}

export function RelationNodeCard({ node, x, y, isRoot }: RelationNodeCardProps) {
  const box = 'absolute flex flex-col gap-1.5 overflow-hidden rounded-lg border px-3 py-2.5 text-left'
  const style = { left: x, top: y, width: NODE_WIDTH, height: NODE_HEIGHT }

  if (node.kind === 'external') {
    return (
      <div className={cn(box, 'border-dashed border-border-strong bg-surface-secondary')} style={style} data-testid="relation-node">
        <div className="flex items-center gap-1.5 min-w-0">
          <GlobeIcon className="w-4 h-4 shrink-0 text-content-tertiary" />
          <span className="truncate text-sm font-medium text-content-primary" title={node.name}>{node.name}</span>
        </div>
        <p className="text-xs text-content-secondary">External stream</p>
        <p className="truncate text-2xs text-content-tertiary" title={node.external?.api_prefix}>{externalLabel(node.external)}</p>
      </div>
    )
  }

  if (node.kind === 'missing') {
    return (
      <div className={cn(box, 'border-dashed border-status-error-border bg-status-error-bg')} style={style} data-testid="relation-node">
        <div className="flex items-center gap-1.5 min-w-0">
          <WarningIcon className="w-4 h-4 shrink-0 text-status-error-text" />
          <span className="truncate text-sm font-medium text-content-primary" title={node.name}>{node.name}</span>
        </div>
        <p className="text-xs text-status-error-text">Not found on this connection</p>
      </div>
    )
  }

  if (node.kind === 'subject') {
    return (
      <div className={cn(box, 'border-dashed border-violet-300 bg-violet-50/50')} style={style} data-testid="relation-node">
        <span className="text-2xs font-semibold uppercase tracking-wide text-violet-700">Subject</span>
        <code className="line-clamp-2 break-words text-xs text-content-primary" title={node.name}>
          {node.name.split('.').map((token, i, tokens) => (
            <Fragment key={i}>
              {i < tokens.length - 1 ? `${token}.` : token}
              {i < tokens.length - 1 && <wbr />}
            </Fragment>
          ))}
        </code>
        <p className="text-2xs text-content-tertiary">No stream captures it</p>
      </div>
    )
  }

  const content = (
    <>
      <div className="flex items-center gap-1.5 min-w-0">
        <span className="truncate text-sm font-semibold text-content-primary" title={node.name}>{node.name}</span>
        {KIND_TAG[node.kind] && (
          <span className="shrink-0 rounded border border-border px-1 text-2xs text-content-tertiary">{KIND_TAG[node.kind]}</span>
        )}
      </div>
      <div className="flex flex-wrap gap-1">
        {streamStats(node).map((stat) => (
          <Stat key={stat}>{stat}</Stat>
        ))}
      </div>
    </>
  )

  if (isRoot) {
    return (
      <div
        className={cn(box, 'border-accent bg-surface-primary shadow-md ring-2 ring-accent-muted')}
        style={style}
        aria-current="page"
        data-testid="relation-node"
      >
        {content}
      </div>
    )
  }

  return (
    <Link
      to={`/streams/${encodeURIComponent(node.name)}/relations`}
      aria-label={`${node.name}: show its relations`}
      className={cn(box, 'border-border-strong bg-surface-primary shadow-sm transition-shadow hover:border-accent hover:shadow-md')}
      style={style}
      draggable={false}
      data-testid="relation-node"
    >
      {content}
    </Link>
  )
}
