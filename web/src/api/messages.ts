import { timestampFromDate } from '@bufbuild/protobuf/wkt'
import { tsToMillis } from '@/utils/timestamp'
import { Direction, SearchStopReason as ProtoSearchStopReason } from '../gen/services/grpc/nats/v1/messages/nats_messages_service_pb'
import type { NatsMessage } from '../gen/types/nats/nats_message_pb'
import type { Message } from '../types/nats'
import { messagesClient } from './grpc/clients'

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
    decoded_unknown_fields: m.decodedUnknownFields || undefined,
    decoded_valid_bytes: m.decodedValidBytes || undefined,
    decoded_auto: m.decodedAuto || undefined,
    decoded_source_id: m.decodedSourceId,
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

export async function getNextMessage(
  connectionId: string,
  streamName: string,
  startSeq: number,
  subjects: string[],
  signal?: AbortSignal,
): Promise<Message | null> {
  const response = await messagesClient.getNextMessage(
    { connectionId, streamName, startSeq: BigInt(startSeq), subjects },
    { signal },
  )
  return response.message ? toMessage(response.message) : null
}

export interface SearchParams {
  connection_id: string
  subject_filter?: string
  direction?: 'forward' | 'backward'
  from_seq?: number
  to_seq?: number
  from_time?: number
  to_time?: number
  cursor_seq?: number
  text?: string
  regex?: boolean
  header_name?: string
  header_value?: string
  max_payload_bytes?: number
}

export type SearchStopReason = 'complete' | 'scan_limit' | 'time_limit' | 'match_limit'

export interface SearchProgress {
  scanned: number
  matched: number
  current_seq: number
  range_first: number
  range_last: number
  resume_seq?: number
}

export interface SearchDone {
  scanned: number
  matched: number
  reason: SearchStopReason
  range_first: number
  range_last: number
  next_seq?: number
}

export type SearchEvent =
  | { kind: 'progress'; progress: SearchProgress }
  | { kind: 'matches'; messages: Message[] }
  | { kind: 'done'; done: SearchDone }

const STOP_REASON: Record<ProtoSearchStopReason, SearchStopReason> = {
  [ProtoSearchStopReason.UNSPECIFIED]: 'complete',
  [ProtoSearchStopReason.COMPLETE]: 'complete',
  [ProtoSearchStopReason.SCAN_LIMIT]: 'scan_limit',
  [ProtoSearchStopReason.TIME_LIMIT]: 'time_limit',
  [ProtoSearchStopReason.MATCH_LIMIT]: 'match_limit',
}

const toBig = (value: number | undefined) => (value != null ? BigInt(value) : undefined)
const toStamp = (ms: number | undefined) => (ms != null ? timestampFromDate(new Date(ms)) : undefined)

export async function* searchMessages(streamName: string, params: SearchParams, signal?: AbortSignal): AsyncGenerator<SearchEvent> {
  const stream = messagesClient.searchMessages(
    {
      connectionId: params.connection_id,
      streamName,
      subjectFilter: params.subject_filter,
      direction:
        params.direction === 'forward' ? Direction.FORWARD : params.direction === 'backward' ? Direction.BACKWARD : Direction.UNSPECIFIED,
      fromSeq: toBig(params.from_seq),
      toSeq: toBig(params.to_seq),
      fromTime: toStamp(params.from_time),
      toTime: toStamp(params.to_time),
      cursorSeq: toBig(params.cursor_seq),
      text: params.text ?? '',
      regex: params.regex ?? false,
      headerName: params.header_name ?? '',
      headerValue: params.header_value ?? '',
      maxPayloadBytes: params.max_payload_bytes,
    },
    { signal },
  )
  for await (const response of stream) {
    const event = response.event
    switch (event.case) {
      case 'progress':
        yield {
          kind: 'progress',
          progress: {
            scanned: Number(event.value.scanned),
            matched: Number(event.value.matched),
            current_seq: Number(event.value.currentSeq),
            range_first: Number(event.value.rangeFirstSeq),
            range_last: Number(event.value.rangeLastSeq),
            resume_seq: event.value.resumeSeq != null ? Number(event.value.resumeSeq) : undefined,
          },
        }
        break
      case 'matches':
        yield { kind: 'matches', messages: event.value.messages.map(toMessage) }
        break
      case 'done':
        yield {
          kind: 'done',
          done: {
            scanned: Number(event.value.scanned),
            matched: Number(event.value.matched),
            reason: STOP_REASON[event.value.reason] ?? 'complete',
            range_first: Number(event.value.rangeFirstSeq),
            range_last: Number(event.value.rangeLastSeq),
            next_seq: event.value.nextSeq != null ? Number(event.value.nextSeq) : undefined,
          },
        }
        break
    }
  }
}
