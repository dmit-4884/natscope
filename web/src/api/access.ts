import type { AccessCheck } from '@/shared/domain/access'
import { AccessStatus, type AccessCheck as ProtoAccessCheck } from '../gen/types/nats/nats_access_pb'

const STATUS: Record<AccessStatus, AccessCheck['status']> = {
  [AccessStatus.UNSPECIFIED]: 'unknown',
  [AccessStatus.ALLOWED]: 'allowed',
  [AccessStatus.DENIED]: 'denied',
}

export function toAccessCheck(check: ProtoAccessCheck | undefined): AccessCheck | undefined {
  if (!check) return undefined
  return {
    status: STATUS[check.status] ?? 'unknown',
    operation: check.operation === 'subscribe' ? 'subscribe' : 'publish',
    subject: check.subject,
  }
}
