import type { Message } from '@/types/nats'
import type { SelectedMessage } from '@/types/messages'

/**
 * Builds the detail-panel selection from a history Message — single source
 * for the row-click path (UnifiedMessageList) and arrow navigation, so both
 * produce identical ids (`history-<seq>`) and field mapping.
 */
export function toSelectedHistoryMessage(msg: Message): SelectedMessage {
  return {
    id: `history-${msg.sequence}`,
    sequence: msg.sequence,
    subject: msg.subject,
    timestamp: msg.timestamp,
    data_base64: msg.data_base64,
    data_size: msg.data_size,
    content_type: msg.content_type,
    headers: msg.headers,
    decoded: msg.decoded ?? undefined,
    decodedType: msg.decoded_type ?? undefined,
    decodeError: msg.decode_error ?? undefined,
    truncated: msg.truncated ?? undefined,
    isLive: false,
  }
}
