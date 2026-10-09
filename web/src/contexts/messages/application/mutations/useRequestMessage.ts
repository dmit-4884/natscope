import { useMutation, useMutationState, type MutationState } from '@tanstack/react-query'
import { requestMessage, type RequestMessageRequest, type RequestReply } from '@/api/publish'

export type { RequestMessageRequest, RequestReply }

type RequestState = MutationState<RequestReply, Error, RequestMessageRequest>

const requestKey = (connectionId: string) => ['request', connectionId] as const

export function useRequestMessage(connectionId: string) {
  return useMutation({
    mutationKey: requestKey(connectionId),
    mutationFn: requestMessage,
    meta: { silent: true },
  })
}

export function useLastRequest(connectionId: string): RequestState | undefined {
  const states = useMutationState({
    filters: { mutationKey: requestKey(connectionId) },
    select: (mutation) => mutation.state as RequestState,
  })
  return states[states.length - 1]
}
