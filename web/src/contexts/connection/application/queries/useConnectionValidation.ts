import { useEffect } from 'react'
import { ConnectError, Code } from '@connectrpc/connect'
import { getErrorReason } from '@/api/errors'
import { logger } from '@/utils/logger'
import { useConnectionHealth } from './useConnectionHealth'

/**
 * Validate a saved connection is reachable; calls onInvalidConnection with the
 * failing error when the server reports it gone, closed, or unavailable.
 * Shares the polled ['health'] query (~100B) with the header indicator.
 */
export function useConnectionValidation(
  connectionId: string | null,
  onInvalidConnection: (error: unknown) => void,
) {
  const { queryError } = useConnectionHealth(connectionId)

  useEffect(() => {
    if (!queryError) return

    const connectError = queryError instanceof ConnectError ? queryError : null
    const reason = getErrorReason(queryError)

    if (
      connectError?.code === Code.NotFound ||
      connectError?.code === Code.Unavailable ||
      reason === 'CONNECTION_NOT_FOUND' ||
      reason === 'NATS_CONNECTION_CLOSED'
    ) {
      logger.warn('Connection validation failed, clearing connection state')
      onInvalidConnection(queryError)
    }
  }, [queryError, onInvalidConnection])
}
