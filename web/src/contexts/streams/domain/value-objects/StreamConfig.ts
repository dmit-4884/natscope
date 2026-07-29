import { ValueObject } from '@/shared'

/** Retention policy. */
export type RetentionPolicy = 'limits' | 'interest' | 'workqueue'

/** Storage type. */
export type StorageType = 'file' | 'memory'

/** Discard policy. */
type DiscardPolicy = 'old' | 'new'

interface StreamConfigProps {
  retention: RetentionPolicy
  maxMessages: number
  maxBytes: number
  maxAge: number // in nanoseconds
  maxConsumers: number
  maxMsgsPerSubject: number
  maxMsgSize: number
  storage: StorageType
  discard: DiscardPolicy
  numReplicas: number
  duplicateWindow: number
  sealed: boolean
  denyDelete: boolean
  denyPurge: boolean
}

/** Stream config with formatting/validation helpers. */
export class StreamConfig extends ValueObject<StreamConfigProps> {
  static fromApi(config: {
    retention: string
    max_msgs: number
    max_bytes: number
    max_age: number
    max_consumers?: number
    max_msgs_per_subject?: number
    max_msg_size?: number
    storage?: string
    discard?: string
    num_replicas?: number
    duplicate_window?: number
    sealed?: boolean
    deny_delete?: boolean
    deny_purge?: boolean
  }): StreamConfig {
    return new StreamConfig({
      retention: config.retention as RetentionPolicy,
      maxMessages: config.max_msgs,
      maxBytes: config.max_bytes,
      maxAge: config.max_age,
      maxConsumers: config.max_consumers ?? -1,
      maxMsgsPerSubject: config.max_msgs_per_subject ?? -1,
      maxMsgSize: config.max_msg_size ?? -1,
      storage: (config.storage ?? 'file') as StorageType,
      discard: (config.discard ?? 'old') as DiscardPolicy,
      numReplicas: config.num_replicas ?? 1,
      duplicateWindow: config.duplicate_window ?? 0,
      sealed: config.sealed ?? false,
      denyDelete: config.deny_delete ?? false,
      denyPurge: config.deny_purge ?? false,
    })
  }

  get retention(): RetentionPolicy {
    return this.props.retention
  }

  get maxMessages(): number {
    return this.props.maxMessages
  }

  get maxBytes(): number {
    return this.props.maxBytes
  }

  get maxAge(): number {
    return this.props.maxAge
  }

  get storage(): StorageType {
    return this.props.storage
  }

  get numReplicas(): number {
    return this.props.numReplicas
  }

  get isSealed(): boolean {
    return this.props.sealed
  }

  /** Human-readable max age. */
  formattedMaxAge(): string {
    if (this.props.maxAge <= 0) {
      return 'Unlimited'
    }

    const seconds = this.props.maxAge / 1_000_000_000
    if (seconds < 60) return `${seconds}s`
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m`
    if (seconds < 86400) return `${Math.floor(seconds / 3600)}h`
    return `${Math.floor(seconds / 86400)}d`
  }

  /** Human-readable max bytes. */
  formattedMaxBytes(): string {
    if (this.props.maxBytes <= 0) {
      return 'Unlimited'
    }

    const bytes = this.props.maxBytes
    if (bytes < 1024) return `${bytes} B`
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
    if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
    return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`
  }

  /** Human-readable max messages. */
  formattedMaxMessages(): string {
    if (this.props.maxMessages <= 0) {
      return 'Unlimited'
    }

    const msgs = this.props.maxMessages
    if (msgs < 1000) return msgs.toString()
    if (msgs < 1_000_000) return `${(msgs / 1000).toFixed(1)}K`
    return `${(msgs / 1_000_000).toFixed(1)}M`
  }

  /** Retention policy display name. */
  retentionDisplayName(): string {
    const names: Record<RetentionPolicy, string> = {
      limits: 'Limits',
      interest: 'Interest',
      workqueue: 'Work Queue',
    }
    return names[this.props.retention]
  }

  /** True if any limit is configured. */
  hasLimits(): boolean {
    return (
      this.props.maxMessages > 0 ||
      this.props.maxBytes > 0 ||
      this.props.maxAge > 0
    )
  }
}
