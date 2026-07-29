import { useEffect, useMemo, useRef, useState } from 'react'
import { useProtoSources } from '@/contexts/proto'
import { useMappingItems } from '@/contexts/mappings'
import { Modal, Button, Dropdown, WarningIcon } from '@/components/ui'
import { plural } from '@/utils/plural'

interface Props {
  onClose: () => void
  onExport: (sourceId: string) => void
  onCopy: (sourceId: string) => Promise<void> | void
  /** If provided, modal opens with this source pre-selected. */
  initialSourceId?: string
}

const ALL_SOURCES = ''

export function ExportMappingsModal({ onClose, onExport, onCopy, initialSourceId = ALL_SOURCES }: Props) {
  const { data: sources = [] } = useProtoSources()
  const { data: items = [] } = useMappingItems()
  const [sourceId, setSourceId] = useState<string>(initialSourceId)
  const [copied, setCopied] = useState(false)
  const [copyError, setCopyError] = useState('')
  const resetTimer = useRef<ReturnType<typeof setTimeout>>()

  useEffect(() => {
    return () => {
      if (resetTimer.current) clearTimeout(resetTimer.current)
    }
  }, [])

  const countsBySource = useMemo(() => {
    const counts = new Map<string, number>()
    for (const m of items) counts.set(m.sourceId, (counts.get(m.sourceId) ?? 0) + 1)
    return counts
  }, [items])

  const options = useMemo(() => {
    const all = [{ value: ALL_SOURCES, label: `All sources (${items.length})` }]
    const perSource = sources.map((s) => {
      const n = countsBySource.get(s.id) ?? 0
      const label = s.sourceType === 'local' ? `${s.name} (local)` : s.name
      return { value: s.id, label: `${label} (${n})` }
    })
    return [...all, ...perSource]
  }, [sources, items.length, countsBySource])

  const selectedCount =
    sourceId === ALL_SOURCES ? items.length : countsBySource.get(sourceId) ?? 0

  const disabled = selectedCount === 0

  const handleDownload = () => {
    onExport(sourceId)
    onClose()
  }

  const handleCopy = async () => {
    try {
      await onCopy(sourceId)
    } catch (err) {
      setCopied(false)
      setCopyError(err instanceof Error ? err.message : 'Failed to copy to clipboard')
      return
    }
    setCopyError('')
    setCopied(true)
    if (resetTimer.current) clearTimeout(resetTimer.current)
    resetTimer.current = setTimeout(() => setCopied(false), 1500)
  }

  return (
    <Modal
      isOpen
      onClose={onClose}
      title="Export Mappings"
      size="md"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button variant="secondary" onClick={handleCopy} disabled={disabled}>
            {copied ? 'Copied' : 'Copy to clipboard'}
          </Button>
          <Button onClick={handleDownload} disabled={disabled}>
            Download
          </Button>
        </>
      }
    >
      <div className="p-6 space-y-4">
        <div className="space-y-2">
          <label className="block text-2xs font-medium text-content-tertiary uppercase tracking-wide">
            Source
          </label>
          <Dropdown
            options={options}
            value={sourceId}
            onChange={setSourceId}
            placeholder="Pick a source…"
          />
          <p className="text-xs text-content-tertiary">
            {selectedCount > 0
              ? `${plural(selectedCount, 'mapping')} will be exported.`
              : 'No mappings match the selected source.'}
          </p>
        </div>
        <p className="text-xs text-content-muted">
          Source IDs are not included in the export — the importing user picks a target source.
        </p>
        {copyError && (
          <div
            data-testid="copy-error"
            className="flex items-start gap-2 px-3 py-2 rounded-button bg-status-error-bg border border-red-200 text-xs text-red-700"
          >
            <WarningIcon className="w-4 h-4 flex-shrink-0 mt-0.5" />
            <span>{copyError}</span>
          </div>
        )}
      </div>
    </Modal>
  )
}
