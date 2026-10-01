import { FramingKind } from '../gen/types/proto/proto_codec_pb'
import type { Framing as PbFraming } from '../gen/types/proto/proto_codec_pb'

export type FramingKindName = 'none' | 'grpc' | 'confluent' | 'varint_delimited' | 'custom'

export interface Framing {
  kind: FramingKindName
  schemaId: number
  prefix: Uint8Array
  suffix: Uint8Array
}

export const NO_FRAMING: Framing = { kind: 'none', schemaId: 0, prefix: new Uint8Array(), suffix: new Uint8Array() }

const KIND_NAMES: Record<FramingKind, FramingKindName> = {
  [FramingKind.UNSPECIFIED]: 'none',
  [FramingKind.GRPC]: 'grpc',
  [FramingKind.CONFLUENT]: 'confluent',
  [FramingKind.VARINT_DELIMITED]: 'varint_delimited',
  [FramingKind.CUSTOM]: 'custom',
}

const KINDS: Record<FramingKindName, FramingKind> = {
  none: FramingKind.UNSPECIFIED,
  grpc: FramingKind.GRPC,
  confluent: FramingKind.CONFLUENT,
  varint_delimited: FramingKind.VARINT_DELIMITED,
  custom: FramingKind.CUSTOM,
}

export function framingFromProto(f: PbFraming | undefined): Framing {
  if (!f) return NO_FRAMING
  return { kind: KIND_NAMES[f.kind], schemaId: f.schemaId, prefix: f.prefix, suffix: f.suffix }
}

export function framingToProto(f: Framing | undefined) {
  if (!f || f.kind === 'none') return undefined
  return { kind: KINDS[f.kind], schemaId: f.schemaId, prefix: f.prefix, suffix: f.suffix }
}
