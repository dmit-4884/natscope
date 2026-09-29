import { useMutation } from '@tanstack/react-query'
import { requestMessage, type RequestMessageRequest, type RequestReply } from '@/api/publish'

export type { RequestMessageRequest, RequestReply }

export function useRequestMessage() {
  return useMutation({
    mutationFn: requestMessage,
    meta: { silent: true },
  })
}
