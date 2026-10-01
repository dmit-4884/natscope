import { ValueObject, Result } from '@/shared'
import { matchSubject } from '@/shared/domain/subjectMatch'

/** Validation error for the Subject value object. */
export class SubjectValidationError extends Error {
  constructor(
    message: string,
    public readonly code: 'EMPTY' | 'INVALID_CHARS' | 'TOO_LONG'
  ) {
    super(message)
    this.name = 'SubjectValidationError'
  }
}

interface SubjectProps {
  value: string
}

/**
 * NATS subject string with pattern matching (`*`/`>`), UUID/numeric-ID
 * normalization, and hierarchy operations.
 */
export class Subject extends ValueObject<SubjectProps> {
  private static readonly UUID_TOKEN = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i
  private static readonly NUMERIC_TOKEN = /^\d+$/
  private static readonly MIN_ID_TOKEN = 8

  /** NATS subject length limit. */
  private static readonly MAX_LENGTH = 256

  private static readonly VALID_CHARS = /^[a-zA-Z0-9._*>-]+$/

  /** Create with validation. */
  static create(value: string): Result<Subject, SubjectValidationError> {
    const trimmed = value.trim()

    if (!trimmed) {
      return Result.err(
        new SubjectValidationError('Subject cannot be empty', 'EMPTY')
      )
    }

    if (trimmed.length > Subject.MAX_LENGTH) {
      return Result.err(
        new SubjectValidationError(
          `Subject exceeds maximum length of ${Subject.MAX_LENGTH}`,
          'TOO_LONG'
        )
      )
    }

    if (!Subject.VALID_CHARS.test(trimmed)) {
      return Result.err(
        new SubjectValidationError(
          'Subject contains invalid characters',
          'INVALID_CHARS'
        )
      )
    }

    return Result.ok(new Subject({ value: trimmed }))
  }

  /** Create without validation; trusted sources only (e.g. API responses). */
  static fromTrusted(value: string): Subject {
    return new Subject({ value })
  }

  get value(): string {
    return this.props.value
  }

  /** Subject to pattern: numeric, UUID and long digit-bearing tokens become '*'; the server learns detected types the same way. */
  toPattern(): string {
    return this.parts()
      .map((t) =>
        Subject.NUMERIC_TOKEN.test(t) ||
        Subject.UUID_TOKEN.test(t) ||
        (t.length >= Subject.MIN_ID_TOKEN && /\d/.test(t))
          ? '*'
          : t,
      )
      .join('.')
  }

  /** Whether this subject matches a NATS pattern (`*` one token, `>` one+). */
  matchesPattern(pattern: string): boolean {
    return matchSubject(this.props.value, pattern)
  }

  /** Hierarchy parts, e.g. 'a.b.c' -> ['a','b','c']. */
  parts(): string[] {
    return this.props.value.split('.')
  }

  /** Parent subject (without the last token), or null. */
  parent(): Subject | null {
    const parts = this.parts()
    if (parts.length <= 1) {
      return null
    }
    return Subject.fromTrusted(parts.slice(0, -1).join('.'))
  }

  /** Last token, e.g. 'orders.created.123' -> '123'. */
  lastToken(): string {
    const parts = this.parts()
    return parts[parts.length - 1]
  }

  /** Depth (number of tokens). */
  depth(): number {
    return this.parts().length
  }

  toString(): string {
    return this.props.value
  }
}
