import type { StreamPurgeRequest } from '@/types/management'

export type PurgeLimitMode = 'all' | 'sequence' | 'keep'

export interface PurgeFormState {
  filter: string
  mode: PurgeLimitMode
  sequence: string
  keep: string
}

export const EMPTY_PURGE_FORM: PurgeFormState = { filter: '', mode: 'all', sequence: '', keep: '' }

export const PURGE_MODE_OPTIONS: ReadonlyArray<{ value: PurgeLimitMode; label: string }> = [
  { value: 'all', label: 'All matching messages' },
  { value: 'sequence', label: 'Up to a sequence' },
  { value: 'keep', label: 'Keep the last N messages' },
]

function parsePositive(raw: string): number | undefined {
  const trimmed = raw.trim()
  if (!/^\d+$/.test(trimmed)) return undefined
  const parsed = Number(trimmed)
  return parsed >= 1 && Number.isSafeInteger(parsed) ? parsed : undefined
}

export function getPurgeFormError(form: PurgeFormState): string | undefined {
  if (form.mode === 'sequence' && parsePositive(form.sequence) === undefined) {
    return 'Enter a sequence number of 1 or higher.'
  }
  if (form.mode === 'keep' && parsePositive(form.keep) === undefined) {
    return 'Enter a message count of 1 or higher.'
  }
  return undefined
}

export function buildPurgeRequest(form: PurgeFormState): StreamPurgeRequest | undefined {
  if (getPurgeFormError(form)) return undefined

  const request: StreamPurgeRequest = {}
  const filter = form.filter.trim()
  if (filter) request.filter = filter
  if (form.mode === 'sequence') request.seq = parsePositive(form.sequence)
  if (form.mode === 'keep') request.keep = parsePositive(form.keep)

  return Object.keys(request).length > 0 ? request : undefined
}
