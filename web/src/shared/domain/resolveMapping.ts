import { matchSubject } from './subjectMatch'

/** Specificity score (higher = more specific): matches the server's
 * natsutil.specificity so client-side previews agree with what Live/Messages
 * actually decode with. */
function patternSpecificity(pattern: string): number {
  let score = 0
  for (const token of pattern.split('.')) {
    if (token === '>') score -= 5
    else if (token === '*') score += 1
    else score += 10
  }
  return score
}

/**
 * Resolves the mapping that would win for a subject, using the same
 * tie-break the server's resolver (natsutil.MappingResolver) applies: an
 * exact (non-wildcard) pattern match wins outright — ties on an identical
 * pattern across sources go to the most recently created one; otherwise the
 * highest-specificity wildcard match wins — ties break by pattern
 * (lexicographic ascending), then by the oldest created.
 *
 * This is the single mapping-resolution algorithm for the frontend: Publish
 * (schema used to encode) and the message viewer (schema used to decode)
 * must agree, or a message can be encoded with one schema and rendered with
 * another (QA-102).
 */
export function resolveMapping<T>(
  subject: string,
  candidates: readonly T[],
  getPattern: (m: T) => string,
  getCreatedAtMs: (m: T) => number,
): T | null {
  let exactBest: T | null = null
  for (const m of candidates) {
    if (getPattern(m) !== subject) continue
    if (!exactBest || getCreatedAtMs(m) >= getCreatedAtMs(exactBest)) {
      exactBest = m
    }
  }
  if (exactBest) return exactBest

  let best: T | null = null
  let bestSpecificity = -Infinity
  for (const m of candidates) {
    const pattern = getPattern(m)
    if (!matchSubject(subject, pattern)) continue

    const specificity = patternSpecificity(pattern)
    if (!best) {
      best = m
      bestSpecificity = specificity
      continue
    }
    if (specificity > bestSpecificity) {
      best = m
      bestSpecificity = specificity
      continue
    }
    if (specificity !== bestSpecificity) continue

    const bestPattern = getPattern(best)
    if (pattern < bestPattern) {
      best = m
    } else if (pattern === bestPattern && getCreatedAtMs(m) < getCreatedAtMs(best)) {
      best = m
    }
  }
  return best
}
