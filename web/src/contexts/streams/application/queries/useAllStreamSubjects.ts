import { useMemo } from 'react'
import { useStreams } from './useStreamList'

/**
 * Flat, deduped, sorted subject patterns across all streams in the active
 * connection, for autocompletes with no stream context (e.g. template editor).
 * Empty when no connection or streams not loaded.
 */
export function useAllStreamSubjects(connectionId: string | null): string[] {
  const { data } = useStreams(connectionId)
  return useMemo(() => {
    const list = data?.streams ?? []
    if (list.length === 0) return []
    const set = new Set<string>()
    for (const s of list) {
      for (const subj of s.subjects ?? []) {
        if (subj) set.add(subj)
      }
    }
    return Array.from(set).sort((a, b) => a.localeCompare(b))
  }, [data])
}
