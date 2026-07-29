import { matchSubject } from '@/shared/domain/subjectMatch'

export interface PatternSegment {
  type: 'static' | 'wildcard'
  value: string
  index?: number
}

export function hasWildcards(pattern: string): boolean {
  return pattern.includes('*') || pattern.includes('>')
}

export function parsePattern(pattern: string): PatternSegment[] {
  const parts = pattern.split('.')
  const segments: PatternSegment[] = []
  let wildcardIndex = 0

  parts.forEach((part, i) => {
    if (part === '*' || part === '>') {
      segments.push({ type: 'wildcard', value: '', index: wildcardIndex++ })
    } else {
      segments.push({ type: 'static', value: part })
    }
    if (i < parts.length - 1) {
      segments.push({ type: 'static', value: '.' })
    }
  })

  return segments
}

export function buildSubject(pattern: string, wildcardValues: string[]): string {
  const parts = pattern.split('.')
  let wildcardIndex = 0
  return parts
    .map((part) => {
      if (part === '*' || part === '>') {
        return wildcardValues[wildcardIndex++] || part
      }
      return part
    })
    .join('.')
}

export function countWildcards(pattern: string): number {
  return pattern.split('.').filter((p) => p === '*' || p === '>').length
}

export function subjectMatchesStream(candidate: string, streamSubjects: string[]): boolean {
  if (!candidate) return false
  return streamSubjects.some((filter) => matchSubject(candidate, filter))
}

/**
 * Inverse of `buildSubject`: recover wildcard slot values from a concrete
 * subject.
 * `*` captures one token, `>` captures the rest (dot-joined); unmatched slots
 * stay empty.
 */
export function extractWildcardValues(pattern: string, subject: string): string[] {
  const patternParts = pattern.split('.')
  const subjectParts = subject.split('.')
  const values: string[] = []

  for (let i = 0; i < patternParts.length; i++) {
    const p = patternParts[i]
    if (p === '>') {
      values.push(subjectParts.length > i ? subjectParts.slice(i).join('.') : '')
      return values
    }
    if (p === '*') {
      values.push(subjectParts[i] ?? '')
    } else if (subjectParts[i] !== p) {
      // Static mismatch: subject wasn't built from this pattern; keep captured,
      // blank the rest.
      const remaining = patternParts.slice(i + 1).filter((t) => t === '*' || t === '>').length
      for (let j = 0; j < remaining; j++) values.push('')
      return values
    }
  }
  return values
}
