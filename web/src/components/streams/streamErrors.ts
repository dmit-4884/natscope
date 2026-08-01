import { Code } from '@connectrpc/connect'
import { getErrorReason, isErrorCode } from '@/api/errors'

export function isStreamNotFound(error: unknown): boolean {
  return getErrorReason(error) === 'NATS_STREAM_NOT_FOUND' || isErrorCode(error, Code.NotFound)
}
