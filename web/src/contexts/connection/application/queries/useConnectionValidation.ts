import { useEffect, useRef } from 'react'
import { ConnectError, Code } from '@connectrpc/connect'
import { getErrorReason } from '@/api/errors'
import { logger } from '@/utils/logger'
import { useConnectionHealth } from './useConnectionHealth'
import { isTransientHealthError } from './connectionHealthError'

const SUSTAINED_FAILURE_MS = 60_000

/**
 * Validate a saved connection is reachable; calls onInvalidConnection with the
 * failing error once the connection is confirmed gone, or a transient outage
 * has not recovered within {@link SUSTAINED_FAILURE_MS}.
 * Shares the polled ['health'] query (~100B) with the header indicator.
 */
export function useConnectionValidation(
  connectionId: string | null,
  onInvalidConnection: (error: unknown) => void,
) {
  const { queryError } = useConnectionHealth(connectionId)
  const transientSinceRef = useRef<number | null>(null)

  useEffect(() => {
    if (!queryError) {
      transientSinceRef.current = null
      return
    }

    const connectError = queryError instanceof ConnectError ? queryError : null
    const reason = getErrorReason(queryError)
    const isGone = connectError?.code === Code.NotFound || reason === 'CONNECTION_NOT_FOUND'

    if (isGone) {
      transientSinceRef.current = null
      logger.warn('Connection validation failed, clearing connection state')
      onInvalidConnection(queryError)
      return
    }

    if (!isTransientHealthError(queryError)) {
      transientSinceRef.current = null
      return
    }

    const since = transientSinceRef.current ?? Date.now()
    transientSinceRef.current = since
    if (Date.now() - since >= SUSTAINED_FAILURE_MS) {
      logger.warn('Connection validation failed, clearing connection state')
      onInvalidConnection(queryError)
    }
  }, [queryError, onInvalidConnection])
}
