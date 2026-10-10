import { Code } from '@connectrpc/connect'
import { isErrorCode } from '@/api/errors'
import { getStreams, getStreamDetail, getStreamNames, getStreamRelations } from '@/api/streams'
import { useConnectionQuery } from '@/hooks/useConnectionQuery'
import { Stream } from '../../domain/entities/Stream'

const STREAM_DETAIL_REFRESH_MS = 5000

export function isRegularStreamName(name: string): boolean {
  return !name.startsWith('KV_') && !name.startsWith('OBJ_')
}

export function filterRegularStreams(streams: Stream[]): Stream[] {
  return streams.filter((s) => isRegularStreamName(s.name.value))
}

export function useStreams(connectionId: string | null) {
  return useConnectionQuery({
    key: ['streams'],
    connectionId,
    fetcher: (signal) => getStreams({ connection_id: connectionId! }, signal),
    refetchOnWindowFocus: true,
  })
}

export function useStreamNames(connectionId: string | null) {
  return useConnectionQuery({
    key: ['streams', 'names'],
    connectionId,
    fetcher: (signal) => getStreamNames(connectionId!, signal),
    refetchOnWindowFocus: true,
  })
}

export function useStreamRelations(connectionId: string | null) {
  return useConnectionQuery({
    key: ['streams', 'relations'],
    connectionId,
    fetcher: (signal) => getStreamRelations(connectionId!, signal),
    staleTime: 30_000,
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
    refetchInterval: (_data, error) => (isErrorCode(error, Code.NotFound) ? false : STREAM_DETAIL_REFRESH_MS),
  })
}
