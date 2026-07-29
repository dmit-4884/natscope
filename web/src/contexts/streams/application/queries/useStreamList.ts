import { getStreams, getStreamDetail } from '@/api/streams'
import { useConnectionQuery } from '@/hooks/useConnectionQuery'
import { Stream } from '../../domain/entities/Stream'

export function filterRegularStreams(streams: Stream[]): Stream[] {
  return streams.filter(
    (s) => !s.name.value.startsWith('KV_') && !s.name.value.startsWith('OBJ_')
  )
}

export function useStreams(connectionId: string | null) {
  return useConnectionQuery({
    key: ['streams'],
    connectionId,
    fetcher: (signal) => getStreams({ connection_id: connectionId! }, signal),
    refetchOnWindowFocus: true,
  })
}

export function useStreamDetail(streamName: string | null, connectionId: string | null) {
  return useConnectionQuery({
    key: ['stream', streamName],
    connectionId,
    fetcher: (signal) => getStreamDetail(streamName!, connectionId!, signal),
    enabled: !!streamName,
    refetchOnWindowFocus: true,
  })
}
