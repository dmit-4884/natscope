import { ConnectError, Code } from '@connectrpc/connect'
import { getErrorReason } from '@/api/errors'

export function isTransientHealthError(error: unknown): boolean {
  if (!(error instanceof ConnectError)) return false
  if (error.code === Code.Unavailable) return true
  return getErrorReason(error) === 'NATS_CONNECTION_CLOSED'
}
