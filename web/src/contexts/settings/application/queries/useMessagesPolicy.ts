import { MessageFetchPolicy } from '../../domain/value-objects/MessageFetchPolicy'
import { useSettings } from './useSettings'

/** Message fetch policy VO; defaults until settings load. */
export function useMessagesPolicy(): MessageFetchPolicy {
  const { data } = useSettings()
  return data?.messages ?? MessageFetchPolicy.default()
}
