import { getErrorReason } from '@/api/errors'

export const REQUEST_TIMEOUT_OPTIONS_MS = [500, 1000, 2000, 5000, 10000, 30000, 60000]

export type RequestFailureKind = 'no-responders' | 'timeout' | 'error'

export interface ReplyTypeCandidate {
  id: string
  fullName: string
  sourceId: string
}

const REQUEST_SUFFIX = 'Request'
const REPLY_SUFFIXES = ['Response', 'Reply', 'Result']

export function formatTimeout(ms: number): string {
  return ms < 1000 ? `${ms} ms` : `${ms / 1000} s`
}

export function literalSubjectError(subject: string): string | null {
  const trimmed = subject.trim()
  if (!trimmed) return null
  if (/\s/.test(trimmed)) return 'A subject cannot contain spaces'
  const tokens = trimmed.split('.')
  if (tokens.some((t) => t === '')) return 'A subject cannot have empty tokens'
  if (tokens.some((t) => t === '*' || t === '>')) return 'Requests need a literal subject without * or > wildcards'
  return null
}

export function literalSubjects(patterns: string[]): string[] {
  return patterns.filter((p) => literalSubjectError(p) === null && p.trim() !== '')
}

export function requestFailureKind(error: unknown): RequestFailureKind {
  switch (getErrorReason(error)) {
    case 'NATS_NO_RESPONDERS':
      return 'no-responders'
    case 'NATS_TIMEOUT':
    case 'DEADLINE_EXCEEDED':
      return 'timeout'
    default:
      return 'error'
  }
}

export function guessReplyType(
  requestType: string | undefined,
  sourceId: string | undefined,
  candidates: ReplyTypeCandidate[],
): string {
  if (!requestType || !sourceId || !requestType.endsWith(REQUEST_SUFFIX)) return ''
  const stem = requestType.slice(0, -REQUEST_SUFFIX.length)
  for (const suffix of REPLY_SUFFIXES) {
    const match = candidates.find((c) => c.sourceId === sourceId && c.fullName === stem + suffix)
    if (match) return match.id
  }
  return ''
}
