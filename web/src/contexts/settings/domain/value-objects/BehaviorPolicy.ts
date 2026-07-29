import { ValueObject, Result, DomainError } from '@/shared'

// Whitelist of destructive ops whose confirm prompt can be disabled.
// Irreversible ops (delete/purge stream, delete bucket) keep type-to-confirm.
export type ConfirmAction =
  | 'deleteConsumer'
  | 'deleteMessage'
  | 'deleteKvKey'
  | 'deleteObject'
  | 'purgeKvHistory'

// Each confirm* flag: true = "ask for confirmation" (safe default). Proto is
// tri-state (nil = confirm); domain collapses nil -> true.
export interface BehaviorPolicyProps {
  confirmDeleteConsumer: boolean
  confirmDeleteMessage: boolean
  confirmDeleteKvKey: boolean
  confirmDeleteObject: boolean
  confirmPurgeKvHistory: boolean
  /** When true, the message-delete dialog pre-selects "secure erase". */
  secureDeleteDefault: boolean
}

/** Loose proto-shaped input — every field optional. */
export interface BehaviorPolicyInput {
  confirmDeleteConsumer?: boolean
  confirmDeleteMessage?: boolean
  confirmDeleteKvKey?: boolean
  confirmDeleteObject?: boolean
  confirmPurgeKvHistory?: boolean
  secureDeleteDefault?: boolean
}

export const BEHAVIOR_POLICY_DEFAULTS: BehaviorPolicyProps = Object.freeze({
  confirmDeleteConsumer: true,
  confirmDeleteMessage: true,
  confirmDeleteKvKey: true,
  confirmDeleteObject: true,
  confirmPurgeKvHistory: true,
  secureDeleteDefault: false,
})

/** Maps a ConfirmAction to its props key. */
export const CONFIRM_ACTION_PROP: Record<ConfirmAction, keyof BehaviorPolicyProps> = {
  deleteConsumer: 'confirmDeleteConsumer',
  deleteMessage: 'confirmDeleteMessage',
  deleteKvKey: 'confirmDeleteKvKey',
  deleteObject: 'confirmDeleteObject',
  purgeKvHistory: 'confirmPurgeKvHistory',
}

export class BehaviorPolicy extends ValueObject<BehaviorPolicyProps> {
  static readonly DEFAULTS = BEHAVIOR_POLICY_DEFAULTS

  private constructor(props: BehaviorPolicyProps) {
    super(props)
  }

  get confirmDeleteConsumer(): boolean {
    return this.props.confirmDeleteConsumer
  }
  get confirmDeleteMessage(): boolean {
    return this.props.confirmDeleteMessage
  }
  get confirmDeleteKvKey(): boolean {
    return this.props.confirmDeleteKvKey
  }
  get confirmDeleteObject(): boolean {
    return this.props.confirmDeleteObject
  }
  get confirmPurgeKvHistory(): boolean {
    return this.props.confirmPurgeKvHistory
  }
  get secureDeleteDefault(): boolean {
    return this.props.secureDeleteDefault
  }

  /** Returns whether the given destructive action should prompt. */
  shouldConfirm(action: ConfirmAction): boolean {
    return this.props[CONFIRM_ACTION_PROP[action]]
  }

  toObject(): BehaviorPolicyProps {
    return { ...this.props }
  }

  merge(patch: BehaviorPolicyInput): Result<BehaviorPolicy, DomainError> {
    return BehaviorPolicy.create({
      confirmDeleteConsumer: patch.confirmDeleteConsumer ?? this.props.confirmDeleteConsumer,
      confirmDeleteMessage: patch.confirmDeleteMessage ?? this.props.confirmDeleteMessage,
      confirmDeleteKvKey: patch.confirmDeleteKvKey ?? this.props.confirmDeleteKvKey,
      confirmDeleteObject: patch.confirmDeleteObject ?? this.props.confirmDeleteObject,
      confirmPurgeKvHistory: patch.confirmPurgeKvHistory ?? this.props.confirmPurgeKvHistory,
      secureDeleteDefault: patch.secureDeleteDefault ?? this.props.secureDeleteDefault,
    })
  }

  static default(): BehaviorPolicy {
    return new BehaviorPolicy({ ...BEHAVIOR_POLICY_DEFAULTS })
  }

  static create(input: Partial<BehaviorPolicyProps>): Result<BehaviorPolicy, DomainError> {
    return Result.ok(new BehaviorPolicy({ ...BEHAVIOR_POLICY_DEFAULTS, ...input }))
  }

  static fromPartial(input?: BehaviorPolicyInput | null): BehaviorPolicy {
    if (!input) return BehaviorPolicy.default()
    return new BehaviorPolicy({
      confirmDeleteConsumer: input.confirmDeleteConsumer ?? BEHAVIOR_POLICY_DEFAULTS.confirmDeleteConsumer,
      confirmDeleteMessage: input.confirmDeleteMessage ?? BEHAVIOR_POLICY_DEFAULTS.confirmDeleteMessage,
      confirmDeleteKvKey: input.confirmDeleteKvKey ?? BEHAVIOR_POLICY_DEFAULTS.confirmDeleteKvKey,
      confirmDeleteObject: input.confirmDeleteObject ?? BEHAVIOR_POLICY_DEFAULTS.confirmDeleteObject,
      confirmPurgeKvHistory: input.confirmPurgeKvHistory ?? BEHAVIOR_POLICY_DEFAULTS.confirmPurgeKvHistory,
      secureDeleteDefault: input.secureDeleteDefault ?? BEHAVIOR_POLICY_DEFAULTS.secureDeleteDefault,
    })
  }
}
