import { ValueObject, Result, DomainError } from '@/shared'

export interface PublishPolicyProps {
  publishTimeoutSec: number
}

export interface PublishPolicyInput {
  publishTimeoutSec?: number
}

const PUBLISH_POLICY_DEFAULTS: PublishPolicyProps = Object.freeze({
  publishTimeoutSec: 10,
})

export class PublishPolicy extends ValueObject<PublishPolicyProps> {
  static readonly DEFAULTS = PUBLISH_POLICY_DEFAULTS

  private constructor(props: PublishPolicyProps) {
    super(props)
  }

  get publishTimeoutSec(): number {
    return this.props.publishTimeoutSec
  }

  toObject(): PublishPolicyProps {
    return { ...this.props }
  }

  merge(patch: PublishPolicyInput): Result<PublishPolicy, DomainError> {
    return PublishPolicy.create({
      publishTimeoutSec: patch.publishTimeoutSec ?? this.props.publishTimeoutSec,
    })
  }

  static default(): PublishPolicy {
    return new PublishPolicy({ ...PUBLISH_POLICY_DEFAULTS })
  }

  static create(input: Partial<PublishPolicyProps>): Result<PublishPolicy, DomainError> {
    const next: PublishPolicyProps = { ...PUBLISH_POLICY_DEFAULTS, ...input }
    if (!Number.isInteger(next.publishTimeoutSec) || next.publishTimeoutSec < 1 || next.publishTimeoutSec > 300) {
      return Result.err(DomainError.validation('publishTimeoutSec must be an integer in [1, 300]', 'publishTimeoutSec'))
    }
    return Result.ok(new PublishPolicy(next))
  }

  static fromPartial(input?: PublishPolicyInput | null): PublishPolicy {
    if (!input) return PublishPolicy.default()
    return new PublishPolicy({
      publishTimeoutSec: input.publishTimeoutSec ?? PUBLISH_POLICY_DEFAULTS.publishTimeoutSec,
    })
  }
}
