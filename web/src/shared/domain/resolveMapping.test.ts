import { describe, it, expect } from 'vitest'
import { resolveMapping } from './resolveMapping'

interface Candidate {
  pattern: string
  sourceId: string
  createdAt: number
}

const c = (pattern: string, sourceId: string, createdAt: number): Candidate => ({ pattern, sourceId, createdAt })
const resolve = (subject: string, candidates: Candidate[]) =>
  resolveMapping(
    subject,
    candidates,
    (m) => m.pattern,
    (m) => m.createdAt,
  )

describe('resolveMapping', () => {
  it('returns null when nothing matches', () => {
    expect(resolve('orders.created', [c('users.*', 'src', 1)])).toBeNull()
  })

  it('an exact literal match beats a wildcard match', () => {
    const exact = c('orders.created', 'src', 1)
    const wildcard = c('orders.*', 'src', 2)
    expect(resolve('orders.created', [wildcard, exact])).toBe(exact)
  })

  it('a more specific wildcard wins', () => {
    const star = c('orders.*', 'src', 1)
    const greater = c('orders.>', 'src', 2)
    expect(resolve('orders.created', [greater, star])).toBe(star)
  })

  // QA-102: the exact repro — the same wildcard pattern bound to two
  // different sources must resolve identically for Publish and the message
  // viewer, and must agree with the server's resolver (natsutil), which
  // breaks identical-pattern wildcard ties by the oldest createdAt.
  it('the same wildcard pattern across two sources resolves to the oldest one, independent of input order', () => {
    const older = c('qa.ui.proto.*', 'qa-ui-local', 1000)
    const newer = c('qa.ui.proto.*', 'qa-ui-files-dup', 2000)

    expect(resolve('qa.ui.proto.dup2', [older, newer])).toBe(older)
    expect(resolve('qa.ui.proto.dup2', [newer, older])).toBe(older)
  })

  it('an identical exact pattern across two sources resolves to the newest one', () => {
    const older = c('orders.created', 'src-a', 1000)
    const newer = c('orders.created', 'src-b', 2000)
    expect(resolve('orders.created', [older, newer])).toBe(newer)
    expect(resolve('orders.created', [newer, older])).toBe(newer)
  })

  it('equal-specificity wildcards tie-break lexicographically by pattern', () => {
    // "*.x" (10 + 1 = 11) and "a.*" (10 + 1 = 11) both match "a.x" with the
    // same specificity; "*.x" sorts first lexicographically ('*' < 'a').
    const starDotX = c('*.x', 'src', 1)
    const aDotStar = c('a.*', 'src', 1)
    expect(resolve('a.x', [aDotStar, starDotX])).toBe(starDotX)
    expect(resolve('a.x', [starDotX, aDotStar])).toBe(starDotX)
  })

  it('equal-specificity, equal-pattern ties break by the oldest createdAt', () => {
    const older = c('orders.*', 'src-a', 100)
    const newer = c('orders.*', 'src-b', 200)
    expect(resolve('orders.created', [newer, older])).toBe(older)
  })
})
