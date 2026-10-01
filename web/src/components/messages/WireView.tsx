import { useState } from 'react'
import type { WireField } from '@/api/decode'
import { useWireDump } from '@/contexts/proto'
import { Alert, ChevronDownIcon, QueryErrorState, SkeletonRows } from '@/components/ui'
import { formatBytes } from '@/utils/formatters'
import { plural } from '@/utils/plural'
import { cn } from '@/utils/cn'

const PREVIEW_BYTES = 32
const TWO_63 = 1n << 63n

function zigzag(v: bigint): bigint {
  return (v >> 1n) ^ -(v & 1n)
}

function signed64(v: bigint): bigint {
  return v >= TWO_63 ? v - (1n << 64n) : v
}

function hex(bytes: Uint8Array): string {
  const shown = Array.from(bytes.slice(0, PREVIEW_BYTES), (b) => b.toString(16).padStart(2, '0')).join(' ')
  return bytes.length > PREVIEW_BYTES ? `${shown} …` : shown
}

function readings(f: WireField): string[] {
  const view = new DataView(new ArrayBuffer(8))
  switch (f.wireType) {
    case 'varint': {
      const out = [f.varint.toString()]
      if (signed64(f.varint) !== f.varint) out.push(`int64 ${signed64(f.varint)}`)
      if (f.varint > 1n) out.push(`sint ${zigzag(f.varint)}`)
      return out
    }
    case 'fixed32':
      view.setUint32(0, Number(f.fixed), true)
      return [f.fixed.toString(), `int32 ${view.getInt32(0, true)}`, `float ${view.getFloat32(0, true)}`]
    case 'fixed64':
      view.setBigUint64(0, f.fixed, true)
      return [f.fixed.toString(), `int64 ${view.getBigInt64(0, true)}`, `double ${view.getFloat64(0, true)}`]
    default:
      return []
  }
}

function FieldRow({ field, depth }: { field: WireField; depth: number }) {
  const nested = field.message.length > 0
  const [open, setOpen] = useState(depth < 2)
  const values = readings(field)

  return (
    <li>
      <div className="flex items-start gap-2 py-1 font-mono text-xs" data-testid="wire-field">
        {nested ? (
          <button
            type="button"
            onClick={() => setOpen((v) => !v)}
            aria-expanded={open}
            aria-label={`${open ? 'Collapse' : 'Expand'} field ${field.number}`}
            className="text-content-muted hover:text-content-primary"
          >
            <ChevronDownIcon className={cn('w-3.5 h-3.5 transition-transform', !open && '-rotate-90')} />
          </button>
        ) : (
          <span className="w-3.5 shrink-0" />
        )}
        <span className="text-accent-text shrink-0">#{field.number}</span>
        <span className="text-content-tertiary shrink-0 w-14">{field.wireType}</span>
        <span className="min-w-0 break-all text-content-primary">
          {values.length > 0 && (
            <>
              {values[0]}
              {values.slice(1).map((v) => (
                <span key={v} className="text-content-tertiary"> · {v}</span>
              ))}
            </>
          )}
          {field.wireType === 'bytes' && !nested && (field.text ? JSON.stringify(field.text) : hex(field.bytes))}
          {field.wireType === 'bytes' && (
            <span className="text-content-tertiary">
              {' '}
              ({nested ? `message, ${plural(field.message.length, 'field')}, ` : ''}
              {formatBytes(field.bytes.length)})
            </span>
          )}
          {nested && field.text && <span className="text-content-tertiary"> · also text {JSON.stringify(field.text)}</span>}
        </span>
        <span className="ml-auto shrink-0 text-2xs text-content-muted tabular-nums">@{field.offset}</span>
      </div>
      {nested && open && (
        <ul className="ml-5 border-l border-border pl-2">
          {field.message.map((child) => (
            <FieldRow key={`${child.offset}-${child.number}`} field={child} depth={depth + 1} />
          ))}
        </ul>
      )}
    </li>
  )
}

interface Props {
  dataBase64: string
  totalBytes: number
}

export function WireView({ dataBase64, totalBytes }: Props) {
  const { data, isLoading, error, refetch } = useWireDump(dataBase64)

  if (isLoading) return <SkeletonRows count={4} />
  if (error) return <QueryErrorState error={error} onRetry={() => void refetch()} />
  if (!data) return null

  return (
    <div className="space-y-3" data-testid="wire-view">
      <p className="text-xs text-content-tertiary">
        Read without a schema: field numbers and wire types only. Bytes that parse as a message open as one.
      </p>
      {data.error && (
        <Alert variant="warning" title={`Stopped after ${formatBytes(data.validBytes)} of ${formatBytes(totalBytes)}`}>
          {data.error}
        </Alert>
      )}
      <ul className="bg-surface-primary border border-border rounded p-2">
        {data.fields.map((f) => (
          <FieldRow key={`${f.offset}-${f.number}`} field={f} depth={0} />
        ))}
      </ul>
    </div>
  )
}
