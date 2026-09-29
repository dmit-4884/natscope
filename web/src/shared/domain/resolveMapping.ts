import { matchSubject } from './subjectMatch'

function patternSpecificity(pattern: string): number {
  let score = 0
  for (const token of pattern.split('.')) {
    if (token === '>') score -= 5
    else if (token === '*') score += 1
    else score += 10
  }
  return score
}

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
