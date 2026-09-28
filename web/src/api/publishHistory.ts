import { tsToMillis } from '@/utils/timestamp'
import { historyClient } from './grpc/clients'

// Legacy interface type kept for existing consumers
export interface PublishHistoryEntry {
  id: string
  connection_id?: string
  connection_url: string
  stream: string
  subject: string
  subject_pattern?: string
  encoding_type: string
  message_type?: string
  payload_json: string
  payload_size: number
  sequence?: number
  success: boolean
  duplicate: boolean
  headers: Record<string, string>
  error?: string
  created_at: number
}

export async function getPublishHistory(
  connectionId?: string,
  connectionUrl?: string,
  stream?: string,
  signal?: AbortSignal,
  pageSize?: number,
): Promise<PublishHistoryEntry[]> {
  const response = await historyClient.listPublishHistory(
    {
      // connectionId is unambiguous regardless of URL count/order; connectionUrl
      // is only sent as a fallback when no id is available. Both
      // filters AND together server-side, so sending both would over-restrict.
      connectionId,
      connectionUrl: connectionId ? undefined : connectionUrl,
      stream,
      pageSize,
    },
    { signal },
  )
  return (response.entries || []).map(item => ({
    id: item.id,
    connection_id: item.connectionId,
    connection_url: item.connectionUrl,
    stream: item.stream,
    subject: item.subject,
    subject_pattern: item.subjectPattern,
    encoding_type: item.encodingType || (item.messageType ? 'protobuf' : 'json'),
    message_type: item.messageType,
    payload_json: item.payloadJson,
    payload_size: item.payloadSize,
    sequence: item.sequence != null ? Number(item.sequence) : undefined,
    success: item.success,
    duplicate: item.duplicate,
    headers: item.headers,
    error: item.error,
    created_at: tsToMillis(item.createdAt),
  }))
}
