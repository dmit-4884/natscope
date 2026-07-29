import type { MessageTemplate } from '@/contexts/templates'
import { downloadBlob } from '@/utils/download'

const EXPORT_VERSION = 1

interface TemplateExportV1 {
  version: 1
  exportedAt: number
  templates: Array<{
    name: string
    subject: string
    messageType: string
    data: string
    headers?: Record<string, string>
    wildcards?: string[]
  }>
}

export type ParsedImport =
  | { ok: true; templates: TemplateExportV1['templates'] }
  | { ok: false; error: string }

/**
 * Build the on-disk export shape. IDs and timestamps are stripped — they're
 * machine-local and would just collide on re-import.
 */
function buildExport(templates: MessageTemplate[]): TemplateExportV1 {
  return {
    version: EXPORT_VERSION,
    exportedAt: Date.now(),
    templates: templates.map((t) => ({
      name: t.name,
      subject: t.subject,
      messageType: t.messageType,
      data: t.data,
      headers: t.headers,
      wildcards: t.wildcards,
    })),
  }
}

export function exportToFile(templates: MessageTemplate[]): void {
  downloadBlob(
    JSON.stringify(buildExport(templates), null, 2),
    `natscope-templates-${new Date().toISOString().split('T')[0]}.json`,
  )
}

export async function exportToClipboard(templates: MessageTemplate[]): Promise<void> {
  await navigator.clipboard.writeText(JSON.stringify(buildExport(templates), null, 2))
}

export function parseImport(text: string): ParsedImport {
  if (!text.trim()) return { ok: false, error: 'Please paste JSON or upload a file.' }
  let parsed: unknown
  try {
    parsed = JSON.parse(text)
  } catch {
    return { ok: false, error: 'Invalid JSON syntax.' }
  }
  if (!parsed || typeof parsed !== 'object') {
    return { ok: false, error: 'Expected an object with a "templates" array.' }
  }
  const obj = parsed as Record<string, unknown>
  if (obj.version !== EXPORT_VERSION) {
    return { ok: false, error: `Unsupported export version: ${String(obj.version)} (expected ${EXPORT_VERSION}).` }
  }
  if (!Array.isArray(obj.templates)) {
    return { ok: false, error: 'Missing "templates" array.' }
  }
  const out: TemplateExportV1['templates'] = []
  for (let i = 0; i < obj.templates.length; i++) {
    const raw = obj.templates[i]
    if (!raw || typeof raw !== 'object') {
      return { ok: false, error: `templates[${i}] is not an object.` }
    }
    const r = raw as Record<string, unknown>
    if (typeof r.name !== 'string' || !r.name.trim()) {
      return { ok: false, error: `templates[${i}].name is required.` }
    }
    if (typeof r.subject !== 'string') {
      return { ok: false, error: `templates[${i}].subject must be a string.` }
    }
    if (typeof r.messageType !== 'string') {
      return { ok: false, error: `templates[${i}].messageType must be a string.` }
    }
    if (typeof r.data !== 'string') {
      return { ok: false, error: `templates[${i}].data must be a string.` }
    }
    let headers: Record<string, string> | undefined
    if (r.headers !== undefined && r.headers !== null) {
      if (typeof r.headers !== 'object') {
        return { ok: false, error: `templates[${i}].headers must be an object.` }
      }
      headers = {}
      for (const [k, v] of Object.entries(r.headers as Record<string, unknown>)) {
        if (typeof v !== 'string') {
          return { ok: false, error: `templates[${i}].headers["${k}"] must be a string.` }
        }
        headers[k] = v
      }
    }
    let wildcards: string[] | undefined
    if (r.wildcards !== undefined && r.wildcards !== null) {
      if (!Array.isArray(r.wildcards)) {
        return { ok: false, error: `templates[${i}].wildcards must be an array.` }
      }
      wildcards = []
      for (let j = 0; j < r.wildcards.length; j++) {
        const w = r.wildcards[j]
        if (typeof w !== 'string') {
          return { ok: false, error: `templates[${i}].wildcards[${j}] must be a string.` }
        }
        wildcards.push(w)
      }
    }
    out.push({ name: r.name, subject: r.subject, messageType: r.messageType, data: r.data, headers, wildcards })
  }
  return { ok: true, templates: out }
}

export async function parseImportFile(file: File): Promise<ParsedImport> {
  return new Promise((resolve) => {
    const reader = new FileReader()
    reader.onload = (e) => resolve(parseImport((e.target?.result as string) ?? ''))
    reader.onerror = () => resolve({ ok: false, error: 'Failed to read file.' })
    reader.readAsText(file)
  })
}
