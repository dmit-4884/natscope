import { getMessage } from '@/api/messages'
import { useConnectionQuery } from '@/hooks/useConnectionQuery'

export function useStreamMessage(connectionId: string | null, streamName: string | null, sequence: number | null) {
  return useConnectionQuery({
    key: ['messages', streamName, 'seq', sequence],
    connectionId,
    enabled: !!streamName && sequence != null && sequence > 0,
    fetcher: () => getMessage(connectionId!, streamName!, sequence!),
    refetchOnWindowFocus: false,
    retry: false,
  })
}
