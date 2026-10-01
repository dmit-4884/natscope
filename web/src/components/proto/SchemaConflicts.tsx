import type { ConflictKind, SchemaConflict, SchemaRef } from '@/api/proto'
import type { ProtoSource } from '@/api/protoSources'
import { Badge } from '@/components/ui'

const KIND_TEXT: Record<ConflictKind, string> = {
  different_shape:
    'Two sources define this type differently. Each mapping names its source, so decoding stays right, but pickers list the type twice.',
  same_shape: 'Two sources define this type identically, usually a shared file vendored twice.',
  file_content: 'Two sources ship different versions of this file.',
}

function Ref({ refTo, sources }: { refTo: SchemaRef; sources: Map<string, string> }) {
  return (
    <li className="min-w-0 truncate">
      <span className="font-medium text-content-primary">{sources.get(refTo.sourceId) ?? refTo.sourceId}</span>
      <span className="font-mono"> · {refTo.file}</span>
      {refTo.revision && <span className="font-mono text-content-muted"> @ {refTo.revision.slice(0, 12)}</span>}
    </li>
  )
}

interface Props {
  conflicts: SchemaConflict[]
  sources: ProtoSource[]
}

export default function SchemaConflicts({ conflicts, sources }: Props) {
  const names = new Map(sources.map((s) => [s.id, s.name]))
  const sorted = [...conflicts].sort(
    (a, b) => Number(a.severity === 'info') - Number(b.severity === 'info') || a.symbol.localeCompare(b.symbol),
  )

  return (
    <ul className="divide-y divide-border" data-testid="schema-conflicts">
      {sorted.map((c) => (
        <li key={`${c.kind}|${c.symbol}|${c.second.sourceId}`} className="py-3 first:pt-0 last:pb-0" data-testid="schema-conflict">
          <div className="flex items-center gap-2 min-w-0">
            <Badge variant={c.severity === 'error' ? 'warning' : 'default'} shape="pill">
              {c.severity === 'error' ? 'Clash' : 'Duplicate'}
            </Badge>
            <span className="font-mono text-sm text-content-primary truncate">{c.symbol}</span>
          </div>
          <p className="mt-1 text-xs text-content-secondary">{KIND_TEXT[c.kind]}</p>
          <ul className="mt-1 text-xs text-content-tertiary space-y-0.5">
            <Ref refTo={c.first} sources={names} />
            <Ref refTo={c.second} sources={names} />
          </ul>
        </li>
      ))}
    </ul>
  )
}
