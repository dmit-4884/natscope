import { tsToMillis } from '@/utils/timestamp'
import type { SubjectMapping as PbMapping } from '../gen/types/mappings/mappings_mapping_pb'
import { framingFromProto, framingToProto, type Framing } from './framing'
import { mappingsClient } from './grpc/clients'

/** Mapping resolvability against current proto sources; mirrors backend enum. */
export type MappingHealth =
  | 'ok'
  | 'source_missing'
  | 'source_disabled'
  | 'selection_missing'
  | 'descriptor_missing'
  | 'type_missing'
  | 'mapping_missing'

export interface MappingHealthInfo {
  id: string
  health: MappingHealth
  detail: string
}

/**
 * Source-aware mapping. Two sources can share a pattern; key by `id` (or
 * `(sourceId, pattern)` for per-source diffs), never by pattern alone.
 */
export interface MappingItem {
  id: string
  pattern: string
  messageType: string
  sourceId: string
  pinnedFingerprint?: string
  framing: Framing
  createdAt: number // Unix milliseconds
  updatedAt: number // Unix milliseconds
}

function toMappingItem(m: PbMapping): MappingItem {
  return {
    id: m.id,
    pattern: m.pattern,
    messageType: m.messageType,
    sourceId: m.sourceId,
    pinnedFingerprint: m.pinnedFingerprint || undefined,
    framing: framingFromProto(m.framing),
    createdAt: tsToMillis(m.createdAt),
    updatedAt: tsToMillis(m.updatedAt),
  }
}

export interface MappingPatch {
  pattern?: string
  messageType?: string
  sourceId?: string
  /** Empty string unpins. */
  pinnedFingerprint?: string
  framing?: Framing
}

export interface MappingsListResult {
  items: MappingItem[]
  /** Total number of mappings across all pages, as reported by the backend. */
  total: number
}

/**
 * Fetches every mapping, walking the cursor; returns backend total + items.
 * `includeTotalCount` on page 0 gives an authoritative count, not items.length.
 */
export async function listMappings(): Promise<MappingsListResult> {
  const items: MappingItem[] = []
  let cursor = ''
  let total = 0
  // Bound the loop in case backend never returns an empty cursor.
  for (let page = 0; page < 100; page++) {
    const response = await mappingsClient.listMappings({
      pageSize: 500,
      pageToken: cursor,
      includeTotalCount: page === 0,
    })
    for (const m of response.mappings) {
      items.push(toMappingItem(m))
    }
    if (page === 0 && response.totalSize !== undefined) {
      total = Number(response.totalSize)
    }
    if (!response.nextPageToken) break
    cursor = response.nextPageToken
  }
  if (total === 0) total = items.length
  return { items, total }
}

export async function createMapping(
  pattern: string,
  messageType: string,
  sourceId: string,
  framing?: Framing,
): Promise<MappingItem> {
  if (!sourceId) {
    throw new Error('createMapping: sourceId is required')
  }
  const response = await mappingsClient.createMapping({ pattern, messageType, sourceId, framing: framingToProto(framing) })
  return toMappingItem(response.mapping!)
}

export async function updateMapping(id: string, patch: MappingPatch): Promise<MappingItem> {
  const response = await mappingsClient.updateMapping({
    id,
    pattern: patch.pattern,
    messageType: patch.messageType,
    sourceId: patch.sourceId,
    pinnedFingerprint: patch.pinnedFingerprint,
    framing: patch.framing ? (framingToProto(patch.framing) ?? {}) : undefined,
  })
  return toMappingItem(response.mapping!)
}

export async function bulkSaveMappings(
  items: Array<{ pattern: string; messageType: string; sourceId: string; framing?: Framing }>,
): Promise<{ created: number; updated: number; deleted: number }> {
  for (const item of items) {
    if (!item.sourceId) {
      throw new Error('bulkSaveMappings: every item must have sourceId')
    }
  }
  const response = await mappingsClient.batchSaveMappings({
    mappings: items.map((item) => ({
      pattern: item.pattern,
      messageType: item.messageType,
      sourceId: item.sourceId,
      framing: framingToProto(item.framing),
    })),
  })
  return {
    created: response.created,
    updated: response.updated,
    deleted: response.deleted,
  }
}

export async function deleteMapping(id: string): Promise<void> {
  await mappingsClient.deleteMapping({ id })
}

/**
 * Resolvability health for the given ids in one call. Use lazily for visible
 * rows — it walks sources/selections/descriptors.
 */
export async function getMappingHealth(ids: string[]): Promise<MappingHealthInfo[]> {
  if (ids.length === 0) return []
  const response = await mappingsClient.batchCheckMappingHealth({ ids })
  return response.items.map((it) => ({
    id: it.id,
    health: it.health as MappingHealth,
    detail: it.detail,
  }))
}
