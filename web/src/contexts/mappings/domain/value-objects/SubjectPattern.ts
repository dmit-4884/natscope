import { ValueObject } from '@/shared'
import { Result } from '@/shared/domain/base/Result'
import { matchSubject } from '@/shared/domain/subjectMatch'

interface SubjectPatternProps {
  value: string
  parts: string[]
  hasWildcards: boolean
}

export type PatternValidationError =
  | 'EMPTY_PATTERN'
  | 'INVALID_CHARACTERS'
  | 'CONSECUTIVE_DOTS'
  | 'TRAILING_DOT'

/**
 * NATS subject pattern: literal segments, `*` (one token), `>` (one+ tokens,
 * must be last).
 */
export class SubjectPattern extends ValueObject<SubjectPatternProps> {
  private constructor(props: SubjectPatternProps) {
    super(props)
  }

  /** Create with validation. */
  static create(value: string): Result<SubjectPattern, PatternValidationError> {
    if (!value || value.trim().length === 0) {
      return Result.err('EMPTY_PATTERN')
    }

    const trimmed = value.trim()

    if (!/^[a-zA-Z0-9_.*>-]+$/.test(trimmed)) {
      return Result.err('INVALID_CHARACTERS')
    }

    if (trimmed.includes('..')) {
      return Result.err('CONSECUTIVE_DOTS')
    }

    if (trimmed.endsWith('.')) {
      return Result.err('TRAILING_DOT')
    }

    const parts = trimmed.split('.')
    const hasWildcards = parts.some((p) => p === '*' || p === '>')

    return Result.ok(
      new SubjectPattern({
        value: trimmed,
        parts,
        hasWildcards,
      })
    )
  }

  /** Create from a trusted source (e.g. database), skipping validation. */
  static fromTrusted(value: string): SubjectPattern {
    const parts = value.split('.')
    const hasWildcards = parts.some((p) => p === '*' || p === '>')
    return new SubjectPattern({ value, parts, hasWildcards })
  }

  get value(): string {
    return this.props.value
  }

  get parts(): ReadonlyArray<string> {
    return this.props.parts
  }

  get hasWildcards(): boolean {
    return this.props.hasWildcards
  }

  /** Whether a subject matches this pattern. */
  matches(subject: string): boolean {
    return matchSubject(subject, this.props.value)
  }

  /** Number of segments. */
  segmentCount(): number {
    return this.props.parts.length
  }

  /** Number of wildcards. */
  wildcardCount(): number {
    return this.props.parts.filter((p) => p === '*' || p === '>').length
  }

  /** Specificity score (higher = more specific) for best-match selection. */
  specificity(): number {
    let score = this.props.parts.length * 10
    for (const part of this.props.parts) {
      if (part === '>') score -= 5
      else if (part === '*') score -= 2
    }
    return score
  }

  /** Whether this pattern is more specific than another. */
  isMoreSpecificThan(other: SubjectPattern): boolean {
    return this.specificity() > other.specificity()
  }

  /** Non-wildcard prefix of the pattern. */
  basePrefix(): string {
    const nonWildcardParts: string[] = []
    for (const part of this.props.parts) {
      if (part === '*' || part === '>') break
      nonWildcardParts.push(part)
    }
    return nonWildcardParts.join('.')
  }

  /** Whether the pattern looks like a concrete subject (contains a UUID). */
  looksLikeSpecificSubject(): boolean {
    const uuidPattern = /[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/i
    return uuidPattern.test(this.props.value)
  }

  toString(): string {
    return this.props.value
  }
}
