import { useEffect, useMemo, useRef } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  bulkCreateTemplates,
  createTemplate,
  deleteAllTemplates,
  deleteTemplate,
  listTemplates,
  updateTemplate,
  type MessageTemplate,
  type TemplateInput,
} from '@/api/templates'
import { safeGetItem, safeSetItem, safeRemoveItem } from '@/utils/safeStorage'

const TEMPLATES_QUERY_KEY = ['templates'] as const

const LEGACY_KEY = 'natscope-templates'
const MIGRATION_FLAG = 'natscope-templates-migrated-to-backend'

interface LegacyTemplate {
  id?: string
  name?: string
  subject?: string
  messageType?: string
  data?: string
  headers?: Record<string, string>
}

/** Reads importable templates from legacy localStorage; null if none. */
function readLegacyTemplates(): TemplateInput[] | null {
  try {
    const raw = safeGetItem(LEGACY_KEY)
    if (!raw) return null
    const parsed = JSON.parse(raw)
    const list: unknown = parsed?.state?.templates ?? parsed?.templates ?? null
    if (!Array.isArray(list) || list.length === 0) return null
    const out: TemplateInput[] = []
    for (const t of list as LegacyTemplate[]) {
      if (typeof t?.name !== 'string' || !t.name.trim()) continue
      out.push({
        name: t.name,
        subject: typeof t.subject === 'string' ? t.subject : '',
        messageType: typeof t.messageType === 'string' ? t.messageType : '',
        data: typeof t.data === 'string' ? t.data : '',
        headers: t.headers && typeof t.headers === 'object' ? t.headers : undefined,
      })
    }
    return out.length > 0 ? out : null
  } catch {
    return null
  }
}

/**
 * Uploads localStorage templates to backend once, then drops the legacy key.
 * Idempotent via a localStorage flag; safe to call from any component.
 */
export function useMigrateLegacyTemplates(): void {
  const ranRef = useRef(false)
  const qc = useQueryClient()

  useEffect(() => {
    if (ranRef.current) return
    ranRef.current = true
    if (safeGetItem(MIGRATION_FLAG) === '1') return
    const legacy = readLegacyTemplates()
    if (!legacy) {
      safeSetItem(MIGRATION_FLAG, '1')
      return
    }
    void (async () => {
      try {
        await bulkCreateTemplates(legacy)
        safeSetItem(MIGRATION_FLAG, '1')
        safeRemoveItem(LEGACY_KEY)
        qc.invalidateQueries({ queryKey: TEMPLATES_QUERY_KEY })
      } catch (e) {
        // Leave flag and legacy key intact so we retry on next load.
        console.warn('[templates] legacy migration failed, will retry on reload:', e)
      }
    })()
  }, [qc])
}

export function useTemplates() {
  return useQuery({
    queryKey: TEMPLATES_QUERY_KEY,
    queryFn: listTemplates,
    staleTime: 30_000,
  })
}

interface SearchContext {
  subject?: string
  messageType?: string
}

export interface TemplateSearchResult {
  contextual: MessageTemplate[]
  rest: MessageTemplate[]
}

const sortByRecent = (a: MessageTemplate, b: MessageTemplate) => b.updatedAt - a.updatedAt

/** Context-aware client-side filter over the cached template list. */
export function useSearchTemplates(query: string, ctx: SearchContext = {}): TemplateSearchResult {
  const { data: templates = [] } = useTemplates()
  return useMemo(() => {
    const q = query.trim().toLowerCase()
    const matched = q
      ? templates.filter(
          (t) =>
            t.name.toLowerCase().includes(q) ||
            t.subject.toLowerCase().includes(q) ||
            t.messageType.toLowerCase().includes(q),
        )
      : [...templates]
    const contextual: MessageTemplate[] = []
    const rest: MessageTemplate[] = []
    for (const t of matched) {
      const matchesCtx =
        (ctx.messageType && t.messageType === ctx.messageType) ||
        (ctx.subject && t.subject === ctx.subject)
      if (matchesCtx) contextual.push(t)
      else rest.push(t)
    }
    contextual.sort(sortByRecent)
    rest.sort(sortByRecent)
    return { contextual, rest }
  }, [templates, query, ctx.subject, ctx.messageType])
}

export function useCreateTemplate() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: createTemplate,
    onSuccess: () => qc.invalidateQueries({ queryKey: TEMPLATES_QUERY_KEY }),
  })
}

export function useUpdateTemplate() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, patch }: { id: string; patch: Partial<TemplateInput> }) => updateTemplate(id, patch),
    onSuccess: () => qc.invalidateQueries({ queryKey: TEMPLATES_QUERY_KEY }),
  })
}

export function useDeleteTemplate() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: deleteTemplate,
    onSuccess: () => qc.invalidateQueries({ queryKey: TEMPLATES_QUERY_KEY }),
  })
}

export function useBulkCreateTemplates() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: bulkCreateTemplates,
    onSuccess: () => qc.invalidateQueries({ queryKey: TEMPLATES_QUERY_KEY }),
  })
}

export function useDeleteAllTemplates() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: deleteAllTemplates,
    onSuccess: () => qc.invalidateQueries({ queryKey: TEMPLATES_QUERY_KEY }),
  })
}

export type { MessageTemplate, TemplateInput }
