import { parseStartDate } from './jumpToTime'
import type { SearchQuery } from './unified/useMessageSearch'

export interface FilterValues {
  subject: string
  startSequence: number | null
  startDate: string | null
  contentFilter: string
  contentRegex: boolean
  header: string
  stopSequence: number | null
  stopDate: string | null
}

const MINUTE_END_MS = 59_999

export const EMPTY_FILTERS: FilterValues = {
  subject: '',
  startSequence: null,
  startDate: null,
  contentFilter: '',
  contentRegex: false,
  header: '',
  stopSequence: null,
  stopDate: null,
}

export function isSearchFilter(filters: FilterValues): boolean {
  return !!filters.contentFilter.trim() || !!filters.header.trim() || filters.stopSequence != null || !!filters.stopDate
}

export function toSearchQuery(filters: FilterValues, listDirection: 'forward' | 'backward'): SearchQuery | null {
  if (!isSearchFilter(filters)) return null

  const startTime = parseStartDate(filters.startDate)
  const stopTime = parseStartDate(filters.stopDate)
  const direction = startTime != null ? 'forward' : listDirection
  const forward = direction === 'forward'
  const [name, ...value] = filters.header.split('=')
  const text = filters.contentFilter.trim()
  const subject = filters.subject.trim()

  const query: SearchQuery = { direction }
  if (subject) query.subject_filter = subject
  if (text) {
    query.text = text
    query.regex = filters.contentRegex
  }
  if (name.trim()) {
    query.header_name = name.trim()
    query.header_value = value.length > 0 ? value.join('=').trim() : undefined
  }
  if (filters.startSequence != null) query[forward ? 'from_seq' : 'to_seq'] = filters.startSequence
  if (filters.stopSequence != null) query[forward ? 'to_seq' : 'from_seq'] = filters.stopSequence
  if (startTime != null) query.from_time = startTime
  if (stopTime != null) {
    if (forward) query.to_time = stopTime + MINUTE_END_MS
    else query.from_time = stopTime
  }
  return query
}
