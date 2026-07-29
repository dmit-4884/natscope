import { useState } from 'react'
import { downloadBlob } from '@/utils/download'
import { useMappingItems, useBulkSaveMappings } from '@/contexts/mappings'

interface Options {
  onImportSuccess?: () => void
}

/**
 * v3 export shape. Source IDs are environment-specific UUIDs and would never
 * resolve cleanly across machines, so we strip them on export and force the
 * importing user to pick a target source per row in the UI.
 */
export interface MappingsExportV3 {
  version: 3
  mappings: Array<{
    pattern: string
    messageType: string
  }>
}

/** Parsed import row, ready to be reviewed and assigned a sourceId in the UI. */
export interface ImportDraftRow {
  pattern: string
  messageType: string
}

export interface ParseResult {
  ok: true
  rows: ImportDraftRow[]
}

export interface ParseError {
  ok: false
  error: string
}

export function useMappingImportExport(opts?: Options) {
  const { data: items = [] } = useMappingItems()
  const bulkSaveMutation = useBulkSaveMappings()
  const [importError, setImportError] = useState('')

  const buildExport = (sourceId?: string): MappingsExportV3 => {
    // Guard against onClick forwarding a SyntheticEvent as the first arg.
    const sid = typeof sourceId === 'string' && sourceId ? sourceId : undefined
    const filtered = sid ? items.filter((m) => m.sourceId === sid) : items
    return {
      version: 3,
      mappings: filtered.map((m) => ({
        pattern: m.pattern,
        messageType: m.messageType,
      })),
    }
  }

  const exportToFile = (sourceId?: string) => {
    downloadBlob(
      JSON.stringify(buildExport(sourceId), null, 2),
      `subject-mappings-${new Date().toISOString().split('T')[0]}.json`,
    )
  }

  const exportToClipboard = async (sourceId?: string) => {
    await navigator.clipboard.writeText(JSON.stringify(buildExport(sourceId), null, 2))
  }

  const parseText = (text: string): ParseResult | ParseError => {
    if (!text.trim()) {
      return { ok: false, error: 'Please paste JSON or upload a file.' }
    }
    let parsed: unknown
    try {
      parsed = JSON.parse(text)
    } catch {
      return { ok: false, error: 'Invalid JSON syntax.' }
    }
    return parseObject(parsed)
  }

  const parseFile = async (file: File): Promise<ParseResult | ParseError> => {
    return new Promise((resolve) => {
      const reader = new FileReader()
      reader.onload = (e) => resolve(parseText((e.target?.result as string) ?? ''))
      reader.onerror = () => resolve({ ok: false, error: 'Failed to read file.' })
      reader.readAsText(file)
    })
  }

  const commit = async (
    rows: Array<{ pattern: string; messageType: string; sourceId: string }>,
  ): Promise<boolean> => {
    setImportError('')
    if (rows.length === 0) {
      setImportError('Nothing to import.')
      return false
    }
    for (const r of rows) {
      if (!r.sourceId) {
        setImportError('Every mapping must have a source assigned.')
        return false
      }
    }
    try {
      await bulkSaveMutation.mutateAsync(rows)
      opts?.onImportSuccess?.()
      return true
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error'
      setImportError(`Failed to save mappings: ${message}`)
      return false
    }
  }

  return {
    items,
    importError,
    setImportError,
    exportToFile,
    exportToClipboard,
    parseText,
    parseFile,
    commit,
    isCommitting: bulkSaveMutation.isPending,
  }
}

function parseObject(parsed: unknown): ParseResult | ParseError {
  if (!parsed || typeof parsed !== 'object') {
    return { ok: false, error: 'Expected a JSON object.' }
  }
  const obj = parsed as { version?: unknown; mappings?: unknown }
  if (obj.version !== 3) {
    return {
      ok: false,
      error: `Unsupported export version. Expected version: 3, got: ${JSON.stringify(obj.version)}.`,
    }
  }
  if (!Array.isArray(obj.mappings)) {
    return { ok: false, error: 'Field "mappings" must be an array.' }
  }
  const rows: ImportDraftRow[] = []
  for (let i = 0; i < obj.mappings.length; i++) {
    const item = obj.mappings[i] as { pattern?: unknown; messageType?: unknown } | null
    if (!item || typeof item !== 'object') {
      return { ok: false, error: `mappings[${i}] is not an object.` }
    }
    const { pattern, messageType } = item
    if (typeof pattern !== 'string' || !pattern) {
      return { ok: false, error: `mappings[${i}].pattern is required.` }
    }
    if (typeof messageType !== 'string' || !messageType) {
      return { ok: false, error: `mappings[${i}].messageType is required.` }
    }
    rows.push({ pattern, messageType })
  }
  if (rows.length === 0) {
    return { ok: false, error: 'No mappings in the file.' }
  }
  return { ok: true, rows }
}
