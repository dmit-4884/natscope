import { useMemo } from 'react'
import { Button, CloseIcon, PlusIcon } from '@/components/ui'
import { useRowKeys } from '@/hooks/useRowKeys'

export interface HeaderEntry {
  key: string
  value: string
}

interface Props {
  headers: HeaderEntry[]
  onAdd: () => void
  onRemove: (index: number) => void
  onUpdate: (index: number, field: 'key' | 'value', value: string) => void
}

/**
 * Standard JetStream publish headers (datalist + hints) — surfaces dedup /
 * optimistic-concurrency features.
 */
const KNOWN_HEADERS: Record<string, { hint: string; valuePlaceholder?: string }> = {
  'Nats-Msg-Id': {
    hint: 'Enables de-duplication: messages with the same id within the stream dedup window are dropped. Tip: use {{uuid}}.',
    valuePlaceholder: '{{uuid}}',
  },
  'Nats-Expected-Stream': {
    hint: 'Publish fails unless the subject is bound to this exact stream.',
  },
  'Nats-Expected-Last-Sequence': {
    hint: 'Optimistic concurrency: publish fails unless the stream last sequence matches this value.',
  },
  'Nats-Expected-Last-Subject-Sequence': {
    hint: 'Optimistic concurrency per subject: publish fails unless the last sequence on this subject matches.',
  },
  'Nats-Expected-Last-Msg-Id': {
    hint: 'Publish fails unless the last message\'s Nats-Msg-Id matches this value.',
  },
  'Nats-Rollup': {
    hint: 'Replaces previous messages: "sub" rolls up the subject, "all" rolls up the whole stream (requires allow_rollup).',
    valuePlaceholder: 'sub',
  },
}

const DATALIST_ID = 'nats-standard-headers'

export function HeadersEditor({ headers, onAdd, onRemove, onUpdate }: Props) {
  const rowKeys = useRowKeys(headers.length)
  const handleAdd = () => {
    rowKeys.registerAdd()
    onAdd()
  }
  const handleRemove = (index: number) => {
    rowKeys.registerRemove(index)
    onRemove(index)
  }

  // Flag duplicate keys: NATS canonicalizes names via textproto, so compare
  // lowercased to catch collisions.
  const duplicateKeys = useMemo(() => {
    const seen = new Map<string, number>()
    for (const h of headers) {
      const k = h.key.trim().toLowerCase()
      if (!k) continue
      seen.set(k, (seen.get(k) ?? 0) + 1)
    }
    return new Set([...seen.entries()].filter(([, n]) => n > 1).map(([k]) => k))
  }, [headers])

  return (
    <div>
      <div className="flex items-center justify-between mb-1">
        <label className="block text-sm font-medium text-gray-700">Headers</label>
        <Button variant="ghost" size="sm" onClick={handleAdd} data-testid="add-header">
          <PlusIcon className="w-4 h-4 mr-1" />
          Add
        </Button>
      </div>
      <datalist id={DATALIST_ID}>
        {Object.keys(KNOWN_HEADERS).map((k) => (
          <option key={k} value={k} />
        ))}
      </datalist>
      {headers.length > 0 && (
        <div className="space-y-2">
          {headers.map((header, index) => {
            const known = KNOWN_HEADERS[header.key.trim()]
            const isDuplicate = duplicateKeys.has(header.key.trim().toLowerCase())
            return (
              <div key={rowKeys.keys[index]}>
                <div className="flex items-center gap-2">
                  <input
                    value={header.key}
                    onChange={(e) => onUpdate(index, 'key', e.target.value)}
                    placeholder="Key"
                    list={DATALIST_ID}
                    className={`flex-1 rounded-md border px-3 py-2 text-sm focus:ring-1 focus:outline-none ${
                      isDuplicate
                        ? 'border-amber-400 bg-status-warning-bg focus:border-status-warning-border focus:ring-status-warning-border'
                        : 'border-border-strong focus:border-border-focus focus:ring-border-focus'
                    }`}
                  />
                  <input
                    value={header.value}
                    onChange={(e) => onUpdate(index, 'value', e.target.value)}
                    placeholder={known?.valuePlaceholder ?? 'Value'}
                    className="flex-1 rounded-md border border-border-strong px-3 py-2 text-sm focus:border-border-focus focus:ring-1 focus:ring-border-focus focus:outline-none"
                  />
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => handleRemove(index)}
                    aria-label={`Remove header ${index + 1}`}
                  >
                    <CloseIcon className="w-4 h-4 text-content-muted hover:text-red-500" />
                  </Button>
                </div>
                {isDuplicate && (
                  <p className="mt-0.5 text-xs text-status-warning-text" data-testid="header-duplicate-warning">
                    Duplicate key — only one value will be sent.
                  </p>
                )}
                {known && !isDuplicate && (
                  <p className="mt-0.5 text-xs text-content-muted" data-testid="header-hint">
                    {known.hint}
                  </p>
                )}
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}
