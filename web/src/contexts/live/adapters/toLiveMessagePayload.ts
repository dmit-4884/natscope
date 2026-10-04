import type { NatsMessage } from '@/gen/types/nats/nats_message_pb'
import { tsToMillis } from '@/utils/timestamp'

export interface LiveMessagePayload {
  stream_name: string
  subject: string
  sequence?: number
  timestamp: number
  data_base64: string
  data_size: number
  content_type?: 'json' | 'text' | 'binary'
  headers?: Record<string, string>
  decoded?: unknown
  decoded_type?: string
  decoded_auto?: boolean
  decoded_source_id?: string
  decode_error?: string
  /** Server capped payload; data_size still holds the original byte count. */
  truncated?: boolean
  reply?: string
}

/** Proto NatsMessage → LiveMessagePayload. */
export function toLiveMessagePayload(msg: NatsMessage, streamName: string): LiveMessagePayload {
  return {
    stream_name: msg.stream || streamName,
    subject: msg.subject,
    sequence: msg.sequence ? Number(msg.sequence) : undefined,
    timestamp: tsToMillis(msg.timestamp),
    data_base64: msg.dataBase64,
    data_size: msg.dataSize,
    content_type: (msg.contentType as 'json' | 'text' | 'binary') || undefined,
    headers: Object.keys(msg.headers).length > 0 ? msg.headers : undefined,
    decoded: msg.decoded ? tryParse(msg.decoded, msg.truncated) : undefined,
    decoded_type: msg.decodedType,
    decoded_auto: msg.decodedAuto || undefined,
    decoded_source_id: msg.decodedSourceId,
    decode_error: msg.decodeError,
    truncated: msg.truncated || undefined,
    reply: msg.reply || undefined,
  }
}

// Recover from byte-truncated JSON tails (truncated=true): parse first, else
// fall back to the raw string so the UI renders the readable prefix.
function tryParse(raw: string, truncated: boolean): unknown {
  try {
    return JSON.parse(raw)
  } catch {
    return truncated ? raw : undefined
  }
}
