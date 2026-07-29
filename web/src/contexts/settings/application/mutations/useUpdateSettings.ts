import { useMutation, useQueryClient, type UseMutationResult } from '@tanstack/react-query'
import { settingsClient } from '@/api/grpc/clients'
import { DomainError } from '@/shared'
import { UserSettings, type UserSettingsUpdate } from '../../domain/entities/UserSettings'
import { toDomainSettings } from '../../adapters/toDomainSettings'
import { toUpdateRequest } from '../../adapters/toUpdateRequest'
import { settingsKeys } from '../queries/useSettings'

/** Applies a settings patch after validating via the aggregate. */
export function useUpdateSettings(): UseMutationResult<
  UserSettings,
  DomainError,
  UserSettingsUpdate
> {
  const queryClient = useQueryClient()

  return useMutation<UserSettings, DomainError, UserSettingsUpdate>({
    mutationFn: async (input): Promise<UserSettings> => {
      // Validate against the current aggregate before sending.
      const current = queryClient.getQueryData<UserSettings>(settingsKeys.detail('current'))
      if (current) {
        const validated = current.applyUpdate(input)
        if (validated.isErr()) {
          throw validated.error
        }
      }
      try {
        const response = await settingsClient.updateSettings(toUpdateRequest(input))
        if (!response.settings) {
          throw DomainError.invariant('Server returned empty settings after update')
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
