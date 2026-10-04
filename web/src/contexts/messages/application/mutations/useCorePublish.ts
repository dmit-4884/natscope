import { useMutation } from '@tanstack/react-query'
import { publishCoreMessage, type CorePublishRequest } from '@/api/publish'

export type { CorePublishRequest }

export function useCorePublish() {
  return useMutation({
    mutationFn: publishCoreMessage,
    meta: { silent: true },
  })
}
