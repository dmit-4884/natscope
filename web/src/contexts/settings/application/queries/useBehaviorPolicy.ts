import { BehaviorPolicy } from '../../domain/value-objects/BehaviorPolicy'
import { useSettings } from './useSettings'

/** Behavior policy VO; defaults (all confirmations ON) until settings load. */
export function useBehaviorPolicy(): BehaviorPolicy {
  const { data } = useSettings()
  return data?.behavior ?? BehaviorPolicy.default()
}
