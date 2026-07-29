/** Canonical domain error, discriminated by `kind` for UI/form mapping. */

export type DomainErrorKind =
  | 'validation'
  | 'invariant'
  | 'not_found'
  | 'conflict'
  | 'unauthorized'
  | 'unknown'

export interface DomainErrorInit {
  kind: DomainErrorKind
  message: string
  field?: string
  reason?: unknown
}

export class DomainError extends Error {
  readonly kind: DomainErrorKind
  readonly field?: string
  readonly reason?: unknown

  constructor(init: DomainErrorInit) {
    super(init.message)
    this.name = 'DomainError'
    this.kind = init.kind
    this.field = init.field
    this.reason = init.reason
    Object.setPrototypeOf(this, DomainError.prototype)
  }

  static validation(message: string, field?: string): DomainError {
    return new DomainError({ kind: 'validation', message, field })
  }

  static invariant(message: string): DomainError {
    return new DomainError({ kind: 'invariant', message })
  }

  static notFound(entity: string): DomainError {
    return new DomainError({ kind: 'not_found', message: `${entity} not found` })
  }

  static conflict(message: string): DomainError {
    return new DomainError({ kind: 'conflict', message })
  }

  static unauthorized(message = 'Unauthorized'): DomainError {
    return new DomainError({ kind: 'unauthorized', message })
  }

  static wrap(err: unknown, kind: DomainErrorKind = 'unknown'): DomainError {
    if (err instanceof DomainError) return err
    if (err instanceof Error) {
      return new DomainError({ kind, message: err.message, reason: err })
    }
    return new DomainError({ kind, message: String(err), reason: err })
  }

  toJSON(): DomainErrorInit {
    return {
      kind: this.kind,
      message: this.message,
      field: this.field,
    }
  }
}
