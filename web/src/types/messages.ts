/** Selected-message shape for the messages UI and messagesViewStore. */
export interface SelectedMessage {
  id: string
  sequence?: number
  subject: string
  timestamp: number
  data_base64: string
  data_size: number
  content_type?: 'json' | 'text' | 'binary'
  headers?: Record<string, string>
  decoded?: unknown
  decodedType?: string
  decodeError?: string
  isLive?: boolean
  /** Preview truncation flag; drives the "Load full payload" affordance. */
  truncated?: boolean
}
