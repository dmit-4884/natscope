import { getNextMessage } from '@/api/messages'
import { useConnectionQuery } from '@/hooks/useConnectionQuery'

export function useNextMessage(
  connectionId: string | null,
  streamName: string | null,
  startSeq: number | null,
  subjects: string[],
) {
  return useConnectionQuery({
    key: ['messages', streamName, 'next', startSeq, subjects],
    connectionId,
    enabled: !!streamName && startSeq != null && startSeq > 0,
    fetcher: (signal) => getNextMessage(connectionId!, streamName!, startSeq!, subjects, signal),
    refetchOnWindowFocus: false,
    retry: false,
  })
}
