import { Subject } from '../domain/value-objects/Subject'

/** Subject pattern via the domain service. */
export function getSubjectPattern(subject: string): string {
  const subjectVO = Subject.fromTrusted(subject)
  return subjectVO.toPattern()
}

/** Whether a subject matches a pattern, via the domain service. */
export function matchesPattern(subject: string, pattern: string): boolean {
  const subjectVO = Subject.fromTrusted(subject)
  return subjectVO.matchesPattern(pattern)
}
