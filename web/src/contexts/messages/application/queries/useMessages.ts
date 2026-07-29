import { getMessages } from '@/api/messages'
import type { GetMessagesParams } from '@/api/messages'
import { useConnectionQuery } from '@/hooks/useConnectionQuery'

export function useMessages(
  streamName: string | null,
  params: Omit<GetMessagesParams, 'connection_id'> & { connection_id: string | null },
  options?: { enabled?: boolean }
) {
  return useConnectionQuery({
    key: [
      'messages',
      streamName,
      params.start_seq ?? null,
      params.start_time ?? null,
      params.limit ?? null,
      params.subject_filter ?? null,
      params.content_filter ?? null,
      params.direction ?? null,
      params.max_payload_bytes ?? null,
    ],
    connectionId: params.connection_id,
    enabled: !!streamName && (options?.enabled ?? true),
    fetcher: (signal) =>
      getMessages(
        streamName!,
        {
          ...params,
          connection_id: params.connection_id!,
        },
        signal,
      ),
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
    refetchOnMount: false,
    staleTime: 5 * 60_000,
    gcTime: 30 * 60_000,
    retry: false,
  })
}
