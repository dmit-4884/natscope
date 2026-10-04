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

export function isSystemPattern(subject: string): boolean {
  return subject.startsWith('$') || subject.startsWith('_')
}

function payloadText(message: LiveMessage): string {
  try {
    return decodeBase64ToUtf8(message.data_base64)
  } catch {
    return ''
  }
}

export function filterReceived(messages: LiveMessage[], query: string, subject?: string | null): LiveMessage[] {
  const q = query.trim().toLowerCase()
  if (!q && !subject) return messages
  return messages.filter((m) => {
    if (subject && m.subject !== subject) return false
    if (!q) return true
    return m.subject.toLowerCase().includes(q) || payloadText(m).toLowerCase().includes(q)
  })
}
