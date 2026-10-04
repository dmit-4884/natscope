type AccessStatus = 'allowed' | 'denied' | 'unknown'

type AccessOperation = 'publish' | 'subscribe'

export interface AccessCheck {
  status: AccessStatus
  operation: AccessOperation
  subject: string
}

export function isDenied(check: AccessCheck | null | undefined): boolean {
  return check?.status === 'denied'
}

export function describePermission(check: Pick<AccessCheck, 'operation' | 'subject'>): string {
  return `${check.operation === 'subscribe' ? 'subscribe to' : 'publish to'} ${check.subject}`
}
