import { useMutation, useQueryClient, type UseMutationResult } from '@tanstack/react-query'
import { settingsClient } from '@/api/grpc/clients'
import { DomainError } from '@/shared'
import { UserSettings } from '../../domain/entities/UserSettings'
import { toDomainSettings } from '../../adapters/toDomainSettings'
import { settingsKeys } from '../queries/useSettings'

/** Resets user settings to server-side defaults. */
export function useResetSettings(): UseMutationResult<UserSettings, DomainError, void> {
  const queryClient = useQueryClient()
  return useMutation<UserSettings, DomainError, void>({
    mutationFn: async (): Promise<UserSettings> => {
      try {
        const response = await settingsClient.resetSettings({})
        if (!response.settings) {
          throw DomainError.invariant('Server returned empty settings after reset')
        }
        return toDomainSettings(response.settings)
      } catch (err) {
        throw DomainError.wrap(err)
      }
    },
    onSuccess: (data) => {
      queryClient.setQueryData(settingsKeys.detail('current'), data)
    },
  })
}
