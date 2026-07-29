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
  error?: string
  created_at: number
}

export async function getPublishHistory(
  connectionUrl?: string,
  stream?: string,
  signal?: AbortSignal,
  pageSize?: number,
): Promise<PublishHistoryEntry[]> {
  const response = await historyClient.listPublishHistory(
    {
      connectionUrl,
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
    error: item.error,
    created_at: tsToMillis(item.createdAt),
  }))
}
