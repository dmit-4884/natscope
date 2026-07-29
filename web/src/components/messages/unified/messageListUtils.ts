import type { Message } from '@/types/nats'
import { decodeBase64ToUtf8 } from '@/utils/base64'

/**
 * Short preview of a base64 payload: tries JSON, then raw text, then
 * "[binary]".
 */
export function getPayloadPreview(dataBase64: string, maxLength = 50): string {
  try {
    const decoded = decodeBase64ToUtf8(dataBase64)
    try {
      const json = JSON.parse(decoded)
      const str = JSON.stringify(json)
      return str.length > maxLength ? str.slice(0, maxLength) + '...' : str
    } catch {
      const text = decoded.slice(0, maxLength)
      return text.length < decoded.length ? text + '...' : text
    }
  } catch {
    return '[binary]'
  }
}

export const LIVE_MESSAGE_LIMITS = [25, 50, 100] as const
export type LiveMessageLimit = (typeof LIVE_MESSAGE_LIMITS)[number]

/** Internal shape used by the live list (same as the adapter output + id). */
export interface LiveMessage {
  id: string
  stream_name: string
  subject: string
  sequence?: number
  timestamp: number
  data_base64: string
  data_size: number
  content_type?: 'json' | 'text' | 'binary'
  headers?: Record<string, string>
  decoded?: unknown
  decodedType?: string
  decodeError?: string
}

export function liveToMessage(m: LiveMessage): Message {
  return {
    sequence: m.sequence ?? 0,
    subject: m.subject,
    timestamp: m.timestamp,
    data_base64: m.data_base64,
    data_size: m.data_size,
    content_type: m.content_type ?? 'binary',
    headers: m.headers,
    decoded: m.decoded as Message['decoded'],
    decoded_type: m.decodedType,
    decode_error: m.decodeError,
  }
}

/**
 * Structural guard for the export path — checks the fields ExportDialog's
 * formatMessage/serializeMessages actually read (subject, data_base64,
 * timestamp, sequence, headers). Malformed entries are dropped instead of
 * exported with undefined/garbage values.
 */
export function isExportableMessage(msg: unknown): msg is Message {
  if (!msg || typeof msg !== 'object') return false
  const m = msg as Record<string, unknown>
  return (
    typeof m.subject === 'string' &&
    typeof m.data_base64 === 'string' &&
    typeof m.timestamp === 'number' &&
    typeof m.sequence === 'number' &&
    (m.headers === undefined ||
      (typeof m.headers === 'object' && m.headers !== null && !Array.isArray(m.headers)))
  )
}

export type WsStatus = 'disconnected' | 'connecting' | 'connected' | 'reconnecting'
export type ViewMode = 'realtime' | 'history'
