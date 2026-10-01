import { useMemo } from 'react'
import type { Framing } from '@/api/framing'
import { useMappingItems } from '@/contexts/mappings'
import { resolveMapping } from '@/shared/domain/resolveMapping'
import type { KVDecodedValue } from '@/types/management'

export interface KVProtoTarget {
  messageType: string
  sourceId: string
  framing?: Framing
  /** Pattern of the matching mapping; unset for a detected type. */
  pattern?: string
  pinnedFingerprint?: string
}

/** Type a key's value encodes as: its $KV.<bucket>.<key> mapping, else the type detected in the stored value. */
export function useKVProtoTarget(bucket: string, key: string, decoded?: KVDecodedValue): KVProtoTarget | null {
  const { data: mappings = [] } = useMappingItems()
  const auto = decoded?.auto ? decoded : undefined
  return useMemo(() => {
    if (!bucket || !key) return null
    const m = resolveMapping(`$KV.${bucket}.${key}`, mappings, (x) => x.pattern, (x) => x.createdAt)
    if (m) {
      return {
        messageType: m.messageType,
        sourceId: m.sourceId,
        framing: m.framing,
        pattern: m.pattern,
        pinnedFingerprint: m.pinnedFingerprint,
      }
    }
    if (auto) return { messageType: auto.messageType, sourceId: auto.sourceId }
    return null
  }, [bucket, key, mappings, auto])
}
