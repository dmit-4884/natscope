import { useQuery } from '@tanstack/react-query'
import { useMemo } from 'react'
import { listSchemaTypes, describeSchemaType, getProtoMessageExample } from '@/api/proto'

export const protoKeys = {
  all: ['proto'] as const,
  types: (sourceId?: string) => [...protoKeys.all, 'types', sourceId ?? ''] as const,
  description: (sourceId: string, fullName: string, reachable: boolean) =>
    [...protoKeys.all, 'description', sourceId, fullName, reachable] as const,
  example: (sourceId: string, fullName: string) => [...protoKeys.all, 'example', sourceId, fullName] as const,
}

export function useSchemaTypes(sourceId?: string) {
  return useQuery({
    queryKey: protoKeys.types(sourceId),
    queryFn: () => listSchemaTypes(sourceId),
  })
}

export function useMessageTypes() {
  const query = useSchemaTypes()
  const messages = useMemo(() => (query.data ?? []).filter((t) => t.kind === 'message'), [query.data])
  return { messages, isLoading: query.isLoading }
}

export function useTypeDescription(sourceId: string | null, fullName: string | null, includeReachable = false) {
  return useQuery({
    queryKey: protoKeys.description(sourceId ?? '', fullName ?? '', includeReachable),
    queryFn: () => describeSchemaType(sourceId!, fullName!, includeReachable),
    enabled: !!sourceId && !!fullName,
  })
}

export function useMessageExample(sourceId: string | null, fullName: string | null) {
  return useQuery({
    queryKey: protoKeys.example(sourceId ?? '', fullName ?? ''),
    queryFn: async () => (await getProtoMessageExample(sourceId!, fullName!)).example,
    enabled: !!sourceId && !!fullName,
  })
}
