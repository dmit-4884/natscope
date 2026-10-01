import { Code, ConnectError } from '@connectrpc/connect'
import { decodeBase64ToBytes } from '@/utils/base64'
import { WireType as PbWireType } from '../gen/types/proto/proto_codec_pb'
import type { WireField as PbWireField } from '../gen/types/proto/proto_codec_pb'
import { framingToProto, type Framing } from './framing'
import { codecClient } from './grpc/clients'

type WireTypeName = 'varint' | 'fixed64' | 'bytes' | 'group' | 'fixed32'

const WIRE_TYPES: Record<PbWireType, WireTypeName> = {
  [PbWireType.UNSPECIFIED]: 'bytes',
  [PbWireType.VARINT]: 'varint',
  [PbWireType.FIXED64]: 'fixed64',
  [PbWireType.BYTES]: 'bytes',
  [PbWireType.GROUP]: 'group',
  [PbWireType.FIXED32]: 'fixed32',
}

export interface UnknownField {
  path: string
  number: number
  wireType: WireTypeName
  size: number
}

export interface WireField {
  number: number
  wireType: WireTypeName
  offset: number
  length: number
  varint: bigint
  fixed: bigint
  bytes: Uint8Array
  text: string
  message: WireField[]
}

export interface WireDump {
  fields: WireField[]
  validBytes: number
  error?: string
}

/** Snapshot-pinned: caller must name the source (same FQN differs per source). */
export interface DecodeRequest {
  data_base64: string
  message_type: string
  source_id: string
  /** Empty uses the source's active schema. */
  schema_fingerprint?: string
  framing?: Framing
}

export interface DecodeResponse {
  success: boolean
  decoded?: unknown
  formatted_json?: string
  error?: string
  unknown_fields?: UnknownField[]
  valid_bytes?: number
}

export async function decodeMessage(request: DecodeRequest): Promise<DecodeResponse> {
  if (!request.source_id) {
    return { success: false, error: 'source_id is required' }
  }
  try {
    const bytes = decodeBase64ToBytes(request.data_base64)
    const response = await codecClient.decodeMessage({
      data: bytes,
      messageType: request.message_type,
      sourceId: request.source_id,
      fingerprint: request.schema_fingerprint || undefined,
      framing: framingToProto(request.framing),
    })

    const result = response.result
    const decoded = result?.data ? JSON.parse(result.data) : undefined
    if (result?.error) {
      return result.validBytes > 0
        ? { success: false, error: result.error, decoded, valid_bytes: result.validBytes }
        : { success: false, error: result.error }
    }
    return {
      success: true,
      decoded,
      formatted_json: result?.data,
      unknown_fields: (result?.unknownFields ?? []).map((f) => ({
        path: f.path,
        number: f.number,
        wireType: WIRE_TYPES[f.wireType],
        size: f.size,
      })),
    }
  } catch (error: unknown) {
    if (error instanceof ConnectError && error.code === Code.ResourceExhausted) {
      return { success: false, error: 'Rate limited - too many requests' }
    }
    throw error
  }
}

function toWireField(f: PbWireField): WireField {
  return {
    number: f.number,
    wireType: WIRE_TYPES[f.wireType],
    offset: f.offset,
    length: f.length,
    varint: f.varint,
    fixed: f.fixed,
    bytes: f.bytes,
    text: f.text,
    message: f.message.map(toWireField),
  }
}

export async function decodeWire(dataBase64: string): Promise<WireDump> {
  const response = await codecClient.decodeWire({ data: decodeBase64ToBytes(dataBase64) })
  return { fields: response.fields.map(toWireField), validBytes: response.validBytes, error: response.error }
}
