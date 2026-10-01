import type { Framing, FramingKindName } from '@/api/framing'
import { bytesToHex, hexToBytes } from '@/utils/hex'

/** Framing as written to an export file, bytes as hex. */
export interface ExportedFraming {
  kind: Exclude<FramingKindName, 'none'>
  schemaId?: number
  prefixHex?: string
  suffixHex?: string
}

const FRAMING_KINDS: FramingKindName[] = ['grpc', 'confluent', 'varint_delimited', 'custom']

export function exportFraming(f: Framing): ExportedFraming | undefined {
  if (f.kind === 'none') return undefined
  return {
    kind: f.kind,
    schemaId: f.schemaId || undefined,
    prefixHex: f.prefix.length ? bytesToHex(f.prefix) : undefined,
    suffixHex: f.suffix.length ? bytesToHex(f.suffix) : undefined,
  }
}

/** Undefined when absent, null when malformed. */
export function importFraming(raw: unknown): Framing | undefined | null {
  if (raw === undefined) return undefined
  const f = raw as Partial<ExportedFraming> | null
  if (!f || typeof f !== 'object' || !FRAMING_KINDS.includes(f.kind as FramingKindName)) return null
  const prefix = hexToBytes(f.prefixHex ?? '')
  const suffix = hexToBytes(f.suffixHex ?? '')
  if (!prefix || !suffix) return null
  return { kind: f.kind as FramingKindName, schemaId: Number(f.schemaId) || 0, prefix, suffix }
}
