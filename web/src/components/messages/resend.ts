import { decodeBase64ToBytes } from '@/utils/base64'
import { countWildcards, extractWildcardValues } from '@/components/streams/publish/subjectPatternUtils'

export interface ResendDraft {
  pattern: string
  wildcards: string[]
  messageJson: string
  headers: { key: string; value: string }[]
  /**
   * Binary (non-UTF-8) with no decoded form — caller MUST block resend to avoid
   * corrupting bytes; `messageJson` is empty when set.
   */
  binaryUndecodable: boolean
}

export interface ResendMessage {
  subject: string
  /** Base64 payload, used when no decoded representation is available. */
  dataBase64?: string
  /**
   * Best decoded representation (server proto-decode, client decode, or parsed
   * JSON).
   */
  decoded?: unknown
  /** Message headers as a plain map. */
  headers?: Record<string, string>
}

/**
 * Build the publish-form draft for resending; strips NATS-reserved (`Nats-*`)
 * headers (server-managed, e.g. Nats-Msg-Id dedup). Pure for testability;
 * caller supplies resolvePattern.
 */
export function buildResendDraft(
  msg: ResendMessage,
  resolvePattern: (subject: string) => string,
): ResendDraft {
  const pattern = resolvePattern(msg.subject)
  const wildcards = countWildcards(pattern) ? extractWildcardValues(pattern, msg.subject) : []
  const payload = resendPayload(msg)
  return {
    pattern,
    wildcards,
    messageJson: payload.binary ? '' : payload.text,
    headers: resendHeaders(msg.headers),
    binaryUndecodable: payload.binary,
  }
}

/**
 * Decode base64 to text with a FATAL TextDecoder — non-UTF-8 throws instead of
 * producing U+FFFD, so binary payloads can't be silently mangled.
 */
function decodeUtf8Strict(base64: string): string {
  return new TextDecoder('utf-8', { fatal: true }).decode(decodeBase64ToBytes(base64))
}

/**
 * Resolve the publish-form payload; returns `binary: true` (no text) when bytes
 * aren't valid UTF-8 and there's no decoded form.
 */
function resendPayload(msg: ResendMessage): { text: string; binary: boolean } {
  if (msg.decoded != null && typeof msg.decoded === 'object') {
    return { text: JSON.stringify(msg.decoded, null, 2), binary: false }
  }
  if (!msg.dataBase64) {
    return { text: '', binary: false }
  }
  let raw: string
  try {
    raw = decodeUtf8Strict(msg.dataBase64)
  } catch {
    // Not valid UTF-8: editing as text would corrupt on resend — signal binary
    // so caller aborts.
    return { text: '', binary: true }
  }
  try {
    return { text: JSON.stringify(JSON.parse(raw), null, 2), binary: false }
  } catch {
    return { text: raw, binary: false }
  }
}

function resendHeaders(headers?: Record<string, string>): { key: string; value: string }[] {
  if (!headers) return []
  return Object.entries(headers)
    .filter(([k]) => !/^nats-/i.test(k))
    .map(([key, value]) => ({ key, value }))
}
