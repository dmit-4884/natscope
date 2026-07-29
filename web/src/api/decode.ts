import { Code, ConnectError } from '@connectrpc/connect'
import { decodeBase64ToBytes } from '@/utils/base64'
import { codecClient } from './grpc/clients'

/** Snapshot-pinned: caller must name the source (same FQN differs per source). */
export interface DecodeRequest {
  data_base64: string
  message_type: string
  source_id: string
  /** Optional. Empty / absent means "use active selection of the source". */
  source_tag?: string
}

export interface DecodeResponse {
  success: boolean
  decoded?: unknown
  formatted_json?: string
  error?: string
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
      tag: request.source_tag || undefined,
    })

    const result = response.result
    if (result?.error) {
      return { success: false, error: result.error }
    }

    const decoded = result?.data ? JSON.parse(result.data) : undefined
    return {
      success: true,
      decoded,
      formatted_json: result?.data,
    }
  } catch (error: unknown) {
    if (error instanceof ConnectError && error.code === Code.ResourceExhausted) {
      return { success: false, error: 'Rate limited - too many requests' }
    }
    throw error
  }
}
