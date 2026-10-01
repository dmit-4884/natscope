import type { FramingKindName } from '@/api/framing'

export const FRAMING_LABELS: Record<FramingKindName, string> = {
  none: 'none',
  grpc: 'gRPC frame',
  confluent: 'Confluent',
  varint_delimited: 'length-delimited',
  custom: 'custom',
}
