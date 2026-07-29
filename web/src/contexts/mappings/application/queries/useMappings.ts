import { useQuery } from '@tanstack/react-query'
import { useMemo, useCallback } from 'react'
import { listMappings } from '@/api/mappings'
import { Mapping } from '../../domain/entities/Mapping'
import { mappingKeys } from './mappingKeys'

/**
 * All mappings as domain entities. UI display only — for runtime decode the
 * backend is authoritative (a pattern can repeat across sources).
 */
function useMappingEntities() {
  // Share the cache slot with `useMappingItems` (key + queryFn) so the two
  // hooks can't race with incompatible shapes. Entity wrapping is client-side.
  const { data: rawItems, isLoading, error, refetch } = useQuery({
    queryKey: mappingKeys.list(),
    queryFn: listMappings,
    select: (result) => result.items,
  })

  const mappings = useMemo(
    () =>
      (rawItems ?? []).map((m) =>
        Mapping.fromApi({
          id: m.id,
          pattern: m.pattern,
          message_type: m.messageType,
          source_id: m.sourceId,
          created_at: m.createdAt,
          updated_at: m.updatedAt,
        }),
      ),
    [rawItems],
  )

  // Sort by (sourceId, pattern) so the table groups by source.
  const sortedMappings = useMemo(
    () =>
      [...mappings].sort((a, b) => {
        if (a.sourceId !== b.sourceId) return a.sourceId.localeCompare(b.sourceId)
        return a.pattern.value.localeCompare(b.pattern.value)
      }),
    [mappings],
  )

  // Get mapping by (pattern, sourceId); pattern alone isn't unique.
  const getMappingByPatternAndSource = useCallback(
    (pattern: string, sourceId: string): Mapping | undefined => {
      return mappings.find((m) => m.pattern.value === pattern && m.sourceId === sourceId)
    },
    [mappings],
  )

  // Best-match mapping for a subject — UI preview only.
  const findMatchingMapping = useCallback(
    (subject: string, sourceFilter?: string): Mapping | null => {
      let bestMatch: Mapping | null = null
      let bestSpecificity = -1

      for (const mapping of mappings) {
        if (sourceFilter && mapping.sourceId !== sourceFilter) continue
        if (mapping.matches(subject)) {
          const specificity = mapping.pattern.specificity()
          if (specificity > bestSpecificity) {
            bestMatch = mapping
            bestSpecificity = specificity
          }
        }
      }

      return bestMatch
    },
    [mappings],
  )

  const filterMappings = useCallback(
    (search: string, sourceFilter?: string): Mapping[] => {
      let result = sortedMappings
      if (sourceFilter) {
        result = result.filter((m) => m.sourceId === sourceFilter)
      }
      if (search) {
        const lower = search.toLowerCase()
        result = result.filter(
          (m) =>
            m.pattern.value.toLowerCase().includes(lower) ||
            m.messageType.toLowerCase().includes(lower) ||
            m.sourceId.toLowerCase().includes(lower),
        )
      }
      return result
    },
    [sortedMappings],
  )

  const stats = useMemo(
    () => ({
      total: mappings.length,
      withWildcards: mappings.filter((m) => m.pattern.hasWildcards).length,
      specificSubjects: mappings.filter((m) => m.looksLikeSpecificSubject()).length,
      sources: new Set(mappings.map((m) => m.sourceId)).size,
    }),
    [mappings],
  )

  return {
    mappings: sortedMappings,
    isLoading,
    error: error as Error | null,
    refetch,
    getMappingByPatternAndSource,
    findMatchingMapping,
    filterMappings,
    stats,
  }
}

/**
 * Best-match mapping for a subject. Pass `sourceFilter` to scope to one source;
 * without it, all sources are considered (UI preview only).
 */
export function useSubjectMappingEntity(subject: string | null, sourceFilter?: string) {
  const { mappings, isLoading } = useMappingEntities()

  const mapping = useMemo(() => {
    if (!subject || mappings.length === 0) return null

    const candidates = sourceFilter ? mappings.filter((m) => m.sourceId === sourceFilter) : mappings

    // Direct match wins.
    const direct = candidates.find((m) => m.pattern.value === subject)
    if (direct) return direct

    // Otherwise best wildcard match by specificity.
    let bestMatch: Mapping | null = null
    let bestSpecificity = -1
    for (const m of candidates) {
      if (m.matches(subject)) {
        const s = m.pattern.specificity()
        if (s > bestSpecificity) {
          bestMatch = m
          bestSpecificity = s
        }
      }
    }
    return bestMatch
  }, [subject, mappings, sourceFilter])

  return {
    mapping,
    messageType: mapping?.messageType ?? null,
    sourceId: mapping?.sourceId ?? null,
    isLoading,
  }
}
