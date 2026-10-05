import { matchSubject } from '@/shared/domain/subjectMatch'
import { decodeBase64ToUtf8 } from '@/utils/base64'
import type { LiveMessage } from '../messages/unified/messageListUtils'

export const SUBJECT_PRESETS = [
  { subject: '$JS.EVENT.ADVISORY.>', label: 'JetStream advisories' },
  { subject: '$JS.EVENT.>', label: 'All JetStream events' },
  { subject: '>', label: 'Everything (busy)' },
] as const

function commonSubjectError(trimmed: string): string | null {
  if (!trimmed) return 'Enter a subject'
  if (/\s/.test(trimmed)) return 'A subject cannot contain spaces'
  // eslint-disable-next-line no-control-regex
  if (/[\u0000-\u001f\u007f]/.test(trimmed)) return 'A subject cannot contain control characters'
  if (trimmed.split('.').some((t) => t === '')) return 'A subject cannot have empty tokens'
  return null
}

export function subscribeSubjectError(subject: string): string | null {
  const trimmed = subject.trim()
  const common = commonSubjectError(trimmed)
  if (common) return common
  const tokens = trimmed.split('.')
  if (tokens.some((t) => t.length > 1 && /[*>]/.test(t))) return 'Wildcards must fill a whole token, like orders.* or orders.>'
  if (tokens.slice(0, -1).includes('>')) return '> must be the last token'
  return null
}

export function publishSubjectError(subject: string): string | null {
  const trimmed = subject.trim()
  const common = commonSubjectError(trimmed)
  if (common) return common
  if (trimmed.split('.').some((t) => t === '*' || t === '>')) return 'Publish to a literal subject, without * or > wildcards'
  return null
}

const payloadTexts = new WeakMap<LiveMessage, string>()

function payloadText(message: LiveMessage): string {
  let text = payloadTexts.get(message)
  if (text === undefined) {
    try {
      text = decodeBase64ToUtf8(message.data_base64).toLowerCase()
    } catch {
      text = ''
    }
    payloadTexts.set(message, text)
  }
  return text
}

export function filterReceived(
  messages: LiveMessage[],
  query: string,
  subject?: string | null,
  muted: string[] = [],
): LiveMessage[] {
  const q = query.trim().toLowerCase()
  if (!q && !subject && muted.length === 0) return messages
  return messages.filter((m) => {
    if (subject && m.subject !== subject) return false
    if (muted.some((pattern) => matchSubject(m.subject, pattern))) return false
    if (!q) return true
    return m.subject.toLowerCase().includes(q) || payloadText(m).includes(q)
  })
}
