import { useQuery } from '@tanstack/react-query'
import { useMemo } from 'react'
import { listSchemaTypes, describeSchemaType, getProtoMessageExample } from '@/api/proto'
import { decodeWire, detectMessageType } from '@/api/decode'

export const protoKeys = {
  all: ['proto'] as const,
  types: (sourceId?: string) => [...protoKeys.all, 'types', sourceId ?? ''] as const,
  description: (sourceId: string, fullName: string, reachable: boolean) =>
    [...protoKeys.all, 'description', sourceId, fullName, reachable] as const,
  example: (sourceId: string, fullName: string) => [...protoKeys.all, 'example', sourceId, fullName] as const,
  wire: (dataBase64: string) => [...protoKeys.all, 'wire', dataBase64] as const,
  candidates: (dataBase64: string) => [...protoKeys.all, 'candidates', dataBase64] as const,
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

export function useWireDump(dataBase64: string) {
  return useQuery({
    queryKey: protoKeys.wire(dataBase64),
    queryFn: () => decodeWire(dataBase64),
    enabled: dataBase64 !== '',
    staleTime: Infinity,
  })
}

export function useTypeCandidates(dataBase64: string, enabled: boolean) {
  return useQuery({
    queryKey: protoKeys.candidates(dataBase64),
    queryFn: () => detectMessageType(dataBase64),
    enabled: enabled && dataBase64 !== '',
  })
}
