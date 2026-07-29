import { timestampFromDate } from '@bufbuild/protobuf/wkt'
import { tsToMillis } from '@/utils/timestamp'
import { Direction } from '../gen/services/grpc/nats/v1/messages/nats_messages_service_pb'
import type { NatsMessage } from '../gen/types/nats/nats_message_pb'
import type { Message } from '../types/nats'
import { messagesClient } from './grpc/clients'

export { Direction } from '../gen/services/grpc/nats/v1/messages/nats_messages_service_pb'

export interface GetMessagesParams {
  connection_id: string
  start_seq?: number
  /**
   * Jump-to-time (epoch ms): server pages from the first message at or after
   * this time. Mutually exclusive with start_seq — pass one, never both.
   */
  start_time?: number
  limit?: number
  subject_filter?: string
  content_filter?: string
  direction?: 'forward' | 'backward'
  /**
   * Preview cap (bytes) per message. 0 = unlimited; unset falls back to
   * settings.messages.maxPayloadBytesInList.
   */
  max_payload_bytes?: number
}

export interface MessagesResponse {
  messages: Message[]
  has_more: boolean
  next_seq: number
}

function toMessage(m: NatsMessage): Message {
  return {
    sequence: Number(m.sequence),
    subject: m.subject,
    timestamp: tsToMillis(m.timestamp),
    data_base64: m.dataBase64,
    data_raw_hex: m.dataRawHex || undefined,
    data_size: m.dataSize,
    content_type: (m.contentType as 'json' | 'text' | 'binary') || 'binary',
    headers: Object.keys(m.headers).length > 0 ? m.headers : undefined,
    decoded: parseDecodedSafe(m.decoded, m.truncated),
    decoded_type: m.decodedType,
    decode_error: m.decodeError,
    truncated: m.truncated || undefined,
  }
}

// Parse the decoded payload; on truncated messages the JSON tail is byte-cut
// and parse throws — fall back to the raw string so the prefix still renders.
function parseDecodedSafe(
  raw: string | undefined,
  _truncated: boolean,
): Record<string, unknown> | string | undefined {
  if (!raw) return undefined
  try {
    return JSON.parse(raw) as Record<string, unknown>
  } catch {
    return raw
  }
}

export async function getMessages(
  streamName: string,
  params: GetMessagesParams,
  signal?: AbortSignal,
): Promise<MessagesResponse> {
  const direction =
    params.direction === 'forward'
      ? Direction.FORWARD
      : params.direction === 'backward'
        ? Direction.BACKWARD
        : Direction.UNSPECIFIED

  const response = await messagesClient.listMessages({
    connectionId: params.connection_id,
    streamName,
    subjectFilter: params.subject_filter,
    contentFilter: params.content_filter,
    direction,
    // Mutually exclusive; start_time wins so we never send both.
    startSeq:
      params.start_time == null && params.start_seq != null ? BigInt(params.start_seq) : undefined,
    startTime: params.start_time != null ? timestampFromDate(new Date(params.start_time)) : undefined,
    limit: params.limit != null ? BigInt(params.limit) : undefined,
    maxPayloadBytes: params.max_payload_bytes,
  }, { signal })

  const messages = response.messages.map(toMessage)

  return {
    messages,
    has_more: response.hasMore,
    next_seq: Number(response.nextSeq),
  }
}

/** Fetches one message in full (no truncation) for the detail panel's "Load full". */
export async function getMessage(
  connectionId: string,
  streamName: string,
  sequence: number,
): Promise<Message> {
  const response = await messagesClient.getMessage({
    connectionId,
    streamName,
    sequence: BigInt(sequence),
  })
  if (!response.message) {
    throw new Error(`message ${sequence} not found in ${streamName}`)
  }
  return toMessage(response.message)
}
