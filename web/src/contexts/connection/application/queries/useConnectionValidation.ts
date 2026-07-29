import { useEffect } from 'react'
import { ConnectError, Code } from '@connectrpc/connect'
import { getHealth } from '@/api/stats'
import { getErrorReason } from '@/api/errors'
import { useConnectionQuery } from '@/hooks/useConnectionQuery'
import { logger } from '@/utils/logger'

/**
 * Validate a saved connection is reachable; calls onInvalidConnection when
 * the server reports it gone, closed, or unavailable. Uses getHealth (~100B)
 */
export function useConnectionValidation(
  connectionId: string | null,
  onInvalidConnection: () => void,
) {
  const { error } = useConnectionQuery({
    key: ['health'],
    connectionId,
    fetcher: (signal) => getHealth(connectionId!, signal),
    retry: false,
    refetchOnWindowFocus: false,
  })

  useEffect(() => {
    if (error) {
      const connectError = error instanceof ConnectError ? error : null
      const reason = getErrorReason(error)

      if (
        connectError?.code === Code.NotFound ||
        connectError?.code === Code.Unavailable ||
        reason === 'CONNECTION_NOT_FOUND' ||
        reason === 'NATS_CONNECTION_CLOSED'
      ) {
        logger.warn('Connection validation failed, clearing connection state')
        onInvalidConnection()
      }
    }
  }, [error, onInvalidConnection])
}
