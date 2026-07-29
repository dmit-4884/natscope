import { useCallback } from 'react'
import {
  CONFIRM_ACTION_PROP,
  type ConfirmAction,
} from '../domain/value-objects/BehaviorPolicy'
import { useBehaviorPolicy } from './queries/useBehaviorPolicy'
import { useUpdateSettings } from './mutations/useUpdateSettings'

export interface Confirmation {
  /** Whether the confirm dialog should show for this action. */
  readonly enabled: boolean
  /** Persists "Don't ask again" (sets confirm toggle false). Idempotent. */
  readonly disable: () => void
}

/**
 * Whether a destructive op should prompt for confirmation. Only whitelisted,
 * recoverable-scope actions (see {@link ConfirmAction}); irreversible-scale
 * ops keep type-to-confirm and never go through this hook.
 */
export function useConfirmation(action: ConfirmAction): Confirmation {
  const behavior = useBehaviorPolicy()
  const updateSettings = useUpdateSettings()
  // Depend on stable `.mutate`, not the whole mutation object (new identity
  // every render), so useCallback memoisation survives.
  const { mutate } = updateSettings

  const disable = useCallback(() => {
    const prop = CONFIRM_ACTION_PROP[action]
    mutate({ behavior: { [prop]: false } })
  }, [action, mutate])

  return {
    enabled: behavior.shouldConfirm(action),
    disable,
  }
}
