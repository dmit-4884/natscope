import { ValueObject, Result } from '@/shared'

/** StreamName validation error. */
export class StreamNameValidationError extends Error {
  constructor(
    message: string,
    public readonly code: 'EMPTY' | 'INVALID_CHARS' | 'TOO_LONG' | 'RESERVED'
  ) {
    super(message)
    this.name = 'StreamNameValidationError'
  }
}

interface StreamNameProps {
  value: string
}

/** Validated NATS JetStream stream name. */
export class StreamName extends ValueObject<StreamNameProps> {
  private static readonly MAX_LENGTH = 256

  // alphanumeric, dash, underscore
  private static readonly VALID_PATTERN = /^[A-Za-z0-9_-]+$/

  private static readonly RESERVED_PREFIXES = ['_', '$']

  /** Validated constructor. */
  static create(value: string): Result<StreamName, StreamNameValidationError> {
    const trimmed = value.trim()

    if (!trimmed) {
      return Result.err(
        new StreamNameValidationError('Stream name cannot be empty', 'EMPTY')
      )
    }

    if (trimmed.length > StreamName.MAX_LENGTH) {
      return Result.err(
        new StreamNameValidationError(
          `Stream name exceeds maximum length of ${StreamName.MAX_LENGTH}`,
          'TOO_LONG'
        )
      )
    }

    if (!StreamName.VALID_PATTERN.test(trimmed)) {
      return Result.err(
        new StreamNameValidationError(
          'Stream name can only contain letters, numbers, dashes, and underscores',
          'INVALID_CHARS'
        )
      )
    }

    if (StreamName.RESERVED_PREFIXES.some((p) => trimmed.startsWith(p))) {
      return Result.err(
        new StreamNameValidationError(
          'Stream name cannot start with reserved prefixes (_ or $)',
          'RESERVED'
        )
      )
    }

    return Result.ok(new StreamName({ value: trimmed }))
  }

  /** Unvalidated constructor for trusted sources. */
  static fromTrusted(value: string): StreamName {
    return new StreamName({ value })
  }

  get value(): string {
    return this.props.value
  }

  /** Uppercase (NATS convention). */
  toUpperCase(): string {
    return this.props.value.toUpperCase()
  }

  toString(): string {
    return this.props.value
  }
}
