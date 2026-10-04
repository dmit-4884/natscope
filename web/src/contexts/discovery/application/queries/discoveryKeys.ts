import { CONNECTION_QUERY_PREFIX } from '@/hooks/useConnectionQuery'

export const discoveryKeys = {
  services: (connectionId: string | null | undefined) =>
    [CONNECTION_QUERY_PREFIX, connectionId ?? null, 'discovery', 'services'] as const,
}
