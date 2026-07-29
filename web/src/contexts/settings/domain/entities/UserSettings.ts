import { AggregateRoot, Result, DomainError } from '@/shared'
import {
  MessageFetchPolicy,
  type MessageFetchPolicyInput,
} from '../value-objects/MessageFetchPolicy'
import {
  LiveSubscriptionPolicy,
  type LiveSubscriptionPolicyInput,
} from '../value-objects/LiveSubscriptionPolicy'
import {
  DisplayPreferences,
  type DisplayPreferencesInput,
} from '../value-objects/DisplayPreferences'
import {
  PublishPolicy,
  type PublishPolicyInput,
} from '../value-objects/PublishPolicy'
import {
  BehaviorPolicy,
  type BehaviorPolicyInput,
} from '../value-objects/BehaviorPolicy'

export interface UserSettingsProps {
  id: string
  messages: MessageFetchPolicy
  live: LiveSubscriptionPolicy
  display: DisplayPreferences
  publish: PublishPolicy
  behavior: BehaviorPolicy
  createdAt: Date
  updatedAt: Date
}

export interface UserSettingsApiDTO {
  id: string
  messages?: MessageFetchPolicyInput | null
  live?: LiveSubscriptionPolicyInput | null
  display?: DisplayPreferencesInput | null
  publish?: PublishPolicyInput | null
  behavior?: BehaviorPolicyInput | null
  createdAt: number
  updatedAt: number
}

// Form-loose update DTO; validated via applyUpdate -> per-VO merge.
export interface UserSettingsUpdate {
  messages?: MessageFetchPolicyInput
  live?: LiveSubscriptionPolicyInput
  display?: DisplayPreferencesInput
  publish?: PublishPolicyInput
  behavior?: BehaviorPolicyInput
}

// Aggregate of policy/preference VOs (each owns its defaults/invariants);
// coordinates immutable updates.
export class UserSettings extends AggregateRoot<UserSettingsProps> {
  private constructor(props: UserSettingsProps) {
    super(props)
  }

  get messages(): MessageFetchPolicy {
    return this.props.messages
  }
  get live(): LiveSubscriptionPolicy {
    return this.props.live
  }
  get display(): DisplayPreferences {
    return this.props.display
  }
  get publish(): PublishPolicy {
    return this.props.publish
  }
  get behavior(): BehaviorPolicy {
    return this.props.behavior
  }
  get createdAt(): Date {
    return this.props.createdAt
  }
  get updatedAt(): Date {
    return this.props.updatedAt
  }

  /** Builds from API DTO; missing fields fall back to VO defaults. */
  static fromApi(dto: UserSettingsApiDTO): UserSettings {
    return new UserSettings({
      id: dto.id,
      messages: MessageFetchPolicy.fromPartial(dto.messages),
      live: LiveSubscriptionPolicy.fromPartial(dto.live),
      display: DisplayPreferences.fromPartial(dto.display),
      publish: PublishPolicy.fromPartial(dto.publish),
      behavior: BehaviorPolicy.fromPartial(dto.behavior),
      createdAt: new Date(dto.createdAt),
      updatedAt: new Date(dto.updatedAt),
    })
  }

  /** Factory for brand new settings (rare — server normally seeds them). */
  static create(id: string): UserSettings {
    const now = new Date()
    return new UserSettings({
      id,
      messages: MessageFetchPolicy.default(),
      live: LiveSubscriptionPolicy.default(),
      display: DisplayPreferences.default(),
      publish: PublishPolicy.default(),
      behavior: BehaviorPolicy.default(),
      createdAt: now,
      updatedAt: now,
    })
  }

  /**
   * Returns a new UserSettings with the patch applied. Any sub-policy
   * validation failure rejects the whole update.
   */
  applyUpdate(input: UserSettingsUpdate): Result<UserSettings, DomainError> {
    const msg = input.messages ? this.props.messages.merge(input.messages) : Result.ok<MessageFetchPolicy, DomainError>(this.props.messages)
    if (msg.isErr()) return Result.err(msg.error)

    const live = input.live ? this.props.live.merge(input.live) : Result.ok<LiveSubscriptionPolicy, DomainError>(this.props.live)
    if (live.isErr()) return Result.err(live.error)

    const display = input.display ? this.props.display.merge(input.display) : Result.ok<DisplayPreferences, DomainError>(this.props.display)
    if (display.isErr()) return Result.err(display.error)

    const publish = input.publish ? this.props.publish.merge(input.publish) : Result.ok<PublishPolicy, DomainError>(this.props.publish)
    if (publish.isErr()) return Result.err(publish.error)

    const behavior = input.behavior ? this.props.behavior.merge(input.behavior) : Result.ok<BehaviorPolicy, DomainError>(this.props.behavior)
    if (behavior.isErr()) return Result.err(behavior.error)

    return Result.ok(
      new UserSettings({
        id: this.props.id,
        messages: msg.value,
        live: live.value,
        display: display.value,
        publish: publish.value,
        behavior: behavior.value,
        createdAt: this.props.createdAt,
        updatedAt: new Date(),
      }),
    )
  }
}
