import { useQuery } from '@tanstack/react-query'
import { useMemo, useCallback } from 'react'
import { getProtoMessages, getProtoMessage } from '@/api/proto'
import { ProtoMessage } from '../../domain/entities/ProtoMessage'

/** Proto query-key factory; sourceId included for source-scoped resources. */
export const protoKeys = {
  all: ['proto'] as const,
  messages: () => [...protoKeys.all, 'messages'] as const,
  message: (sourceId: string, name: string) => [...protoKeys.all, 'message', sourceId, name] as const,
  example: (sourceId: string, name: string) => [...protoKeys.all, 'example', sourceId, name] as const,
}

/** All proto messages from active snapshots, grouped by (sourceId, package). */
export function useProtoMessageEntities() {
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: protoKeys.messages(),
    queryFn: async () => {
      const response = await getProtoMessages()
      return {
        messages: response.messages.map((m) => ProtoMessage.fromApi(m)),
        groupedBySourcePackage: response.grouped_by_source_package,
      }
    },
  })

  const messages = useMemo(() => data?.messages ?? [], [data])
  const groupedBySourcePackage = useMemo(
    () => data?.groupedBySourcePackage ?? {},
    [data],
  )

  /** Sources sorted alphabetically. */
  const sources = useMemo(() => {
    const seen = new Set<string>()
    for (const m of messages) seen.add(m.sourceId)
    return Array.from(seen).sort()
  }, [messages])

  /** Packages sorted alphabetically (across all sources). */
  const packages = useMemo(() => {
    const seen = new Set<string>()
    for (const m of messages) seen.add(m.packageName)
    return Array.from(seen).sort()
  }, [messages])

  // Find by (sourceId, fullName); FQN alone isn't unique across sources.
  const getMessageByName = useCallback(
    (sourceId: string, fullName: string): ProtoMessage | undefined => {
      return messages.find((m) => m.sourceId === sourceId && m.fullName === fullName)
    },
    [messages],
  )

  /** Filter by free-text search; optionally scope to a source. */
  const filterMessages = useCallback(
    (search: string, sourceFilter?: string): ProtoMessage[] => {
      let result = messages
      if (sourceFilter) result = result.filter((m) => m.sourceId === sourceFilter)
      if (search) {
        const lower = search.toLowerCase()
        result = result.filter(
          (m) =>
            m.fullName.toLowerCase().includes(lower) ||
            m.shortName().toLowerCase().includes(lower) ||
            m.sourceId.toLowerCase().includes(lower),
        )
      }
      return result
    },
    [messages],
  )

  /** Messages within a specific (sourceId, package) cell. */
  const getMessagesBySourceAndPackage = useCallback(
    (sourceId: string, packageName: string): ProtoMessage[] => {
      return messages.filter((m) => m.sourceId === sourceId && m.packageName === packageName)
    },
    [messages],
  )

  return {
    messages,
    sources,
    packages,
    groupedBySourcePackage,
    isLoading,
    error: error as Error | null,
    refetch,
    getMessageByName,
    filterMessages,
    getMessagesBySourceAndPackage,
  }
}

/** Single proto message scoped to a source snapshot. */
export function useProtoMessageEntity(sourceId: string | null, messageName: string | null) {
  const enabled = !!sourceId && !!messageName
  const { data, isLoading, error } = useQuery({
    queryKey: protoKeys.message(sourceId ?? '', messageName ?? ''),
    queryFn: async () => {
      const response = await getProtoMessage(sourceId!, messageName!)
      return ProtoMessage.fromApi(response)
    },
    enabled,
  })

  return {
    message: data ?? null,
    isLoading,
    error: error as Error | null,
  }
}

