type AccessStatus = 'allowed' | 'denied' | 'unknown'

type AccessOperation = 'publish' | 'subscribe'

export interface AccessCheck {
  status: AccessStatus
  operation: AccessOperation
  subject: string
}

export class AccessDeniedError extends Error {
  readonly access: AccessCheck

  constructor(message: string, access: AccessCheck) {
    super(message)
    this.name = 'AccessDeniedError'
    this.access = access
    Object.setPrototypeOf(this, AccessDeniedError.prototype)
  }
}

export function isDenied(check: AccessCheck | null | undefined): boolean {
  return check?.status === 'denied'
}

export function describePermission(check: Pick<AccessCheck, 'operation' | 'subject'>): string {
  return `${check.operation === 'subscribe' ? 'subscribe to' : 'publish to'} ${check.subject}`
}
