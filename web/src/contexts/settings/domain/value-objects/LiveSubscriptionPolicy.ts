import { ValueObject, Result, DomainError } from '@/shared'

export type SubscriptionMode = 'core_nats' | 'jetstream_ordered'

// deliverPolicy, batch/buffer tuning are hardcoded server-side (removed).
// core_nats is the only mode that works without create_consumer permission.
export interface LiveSubscriptionPolicyProps {
  subscriptionMode: SubscriptionMode
  maxDisplayRate: number
}

export interface LiveSubscriptionPolicyInput {
  subscriptionMode?: string
  maxDisplayRate?: number
}

const VALID_MODES: readonly SubscriptionMode[] = ['core_nats', 'jetstream_ordered']

const LIVE_SUBSCRIPTION_POLICY_DEFAULTS: LiveSubscriptionPolicyProps = Object.freeze({
  subscriptionMode: 'core_nats',
  maxDisplayRate: 0,
})

export class LiveSubscriptionPolicy extends ValueObject<LiveSubscriptionPolicyProps> {
  static readonly DEFAULTS = LIVE_SUBSCRIPTION_POLICY_DEFAULTS

  private constructor(props: LiveSubscriptionPolicyProps) {
    super(props)
  }

  get subscriptionMode(): SubscriptionMode {
    return this.props.subscriptionMode
  }
  get maxDisplayRate(): number {
    return this.props.maxDisplayRate
  }

  toObject(): LiveSubscriptionPolicyProps {
    return { ...this.props }
  }

  merge(patch: LiveSubscriptionPolicyInput): Result<LiveSubscriptionPolicy, DomainError> {
    return LiveSubscriptionPolicy.create({
      subscriptionMode: (patch.subscriptionMode ?? this.props.subscriptionMode) as SubscriptionMode,
      maxDisplayRate: patch.maxDisplayRate ?? this.props.maxDisplayRate,
    })
  }

  static default(): LiveSubscriptionPolicy {
    return new LiveSubscriptionPolicy({ ...LIVE_SUBSCRIPTION_POLICY_DEFAULTS })
  }

  static create(
    input: Partial<LiveSubscriptionPolicyProps>,
  ): Result<LiveSubscriptionPolicy, DomainError> {
    const next: LiveSubscriptionPolicyProps = {
      ...LIVE_SUBSCRIPTION_POLICY_DEFAULTS,
      ...input,
    }
    if (!VALID_MODES.includes(next.subscriptionMode)) {
      return Result.err(DomainError.validation('subscriptionMode must be core_nats or jetstream_ordered', 'subscriptionMode'))
    }
    if (!Number.isInteger(next.maxDisplayRate) || next.maxDisplayRate < 0 || next.maxDisplayRate > 10_000) {
      return Result.err(DomainError.validation('maxDisplayRate must be an integer in [0, 10000]', 'maxDisplayRate'))
    }
    return Result.ok(new LiveSubscriptionPolicy(next))
  }

  static fromPartial(input?: LiveSubscriptionPolicyInput | null): LiveSubscriptionPolicy {
    if (!input) return LiveSubscriptionPolicy.default()
    const mode = (input.subscriptionMode ?? LIVE_SUBSCRIPTION_POLICY_DEFAULTS.subscriptionMode) as SubscriptionMode
    return new LiveSubscriptionPolicy({
      subscriptionMode: VALID_MODES.includes(mode) ? mode : LIVE_SUBSCRIPTION_POLICY_DEFAULTS.subscriptionMode,
      maxDisplayRate: input.maxDisplayRate ?? LIVE_SUBSCRIPTION_POLICY_DEFAULTS.maxDisplayRate,
    })
  }
}
