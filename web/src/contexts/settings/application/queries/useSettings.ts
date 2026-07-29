import { useQuery, type UseQueryResult } from '@tanstack/react-query'
import { settingsClient } from '@/api/grpc/clients'
import { createQueryKeys } from '@/shared'
import { UserSettings } from '../../domain/entities/UserSettings'
import { toDomainSettings } from '../../adapters/toDomainSettings'

export const settingsKeys = createQueryKeys('settings')

/** Loads user settings as a domain entity; defaults if server unseeded. */
export function useSettings(): UseQueryResult<UserSettings, Error> {
  return useQuery({
    queryKey: settingsKeys.detail('current'),
    queryFn: async (): Promise<UserSettings> => {
      const response = await settingsClient.getSettings({})
      if (!response.settings) {
        return UserSettings.create('default')
      }
      return toDomainSettings(response.settings)
    },
    staleTime: 30_000,
  })
}
