import { LiveSubscriptionPolicy } from '../../domain/value-objects/LiveSubscriptionPolicy'
import { useSettings } from './useSettings'

/** Live subscription policy VO; defaults until settings load. */
export function useLivePolicy(): LiveSubscriptionPolicy {
  const { data } = useSettings()
  return data?.live ?? LiveSubscriptionPolicy.default()
}
