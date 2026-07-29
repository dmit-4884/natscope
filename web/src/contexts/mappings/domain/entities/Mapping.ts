import { Entity } from '@/shared'
import { SubjectPattern } from '../value-objects/SubjectPattern'

export type MappingHealth =
  | 'ok'
  | 'source_missing'
  | 'source_disabled'
  | 'selection_missing'
  | 'descriptor_missing'
  | 'type_missing'

interface MappingProps {
  id: string
  pattern: SubjectPattern
  messageType: string
  /** Source the mapping is bound to (required); a pattern can repeat across
   *  sources. */
  sourceId: string
  createdAt: Date
  updatedAt: Date | null
  /** Optional. Computed lazily via mappings.HealthBatch RPC. */
  health?: MappingHealth
  healthDetail?: string
}

/**
 * Subject-to-message-type mapping bound to one proto source. Decoding happens
 * within that source's active snapshot — no merged cross-source registry.
 */
export class Mapping extends Entity<MappingProps> {
  private constructor(props: MappingProps) {
    super(props)
  }

  static fromApi(data: {
    id: string
    pattern: string
    message_type: string
    source_id: string
    created_at: number
    updated_at?: number
  }): Mapping {
    return new Mapping({
      id: data.id,
      pattern: SubjectPattern.fromTrusted(data.pattern),
      messageType: data.message_type,
      sourceId: data.source_id,
      createdAt: new Date(data.created_at),
      updatedAt: data.updated_at ? new Date(data.updated_at) : null,
    })
  }

  static createNew(pattern: string, messageType: string, sourceId: string): Mapping {
    if (!sourceId) {
      throw new Error('Mapping.createNew: sourceId is required')
    }
    return new Mapping({
      id: '',
      pattern: SubjectPattern.fromTrusted(pattern),
      messageType,
      sourceId,
      createdAt: new Date(),
      updatedAt: null,
    })
  }

  withHealth(health: MappingHealth, detail?: string): Mapping {
    return new Mapping({
      ...this.props,
      health,
      healthDetail: detail,
    })
  }

  get pattern(): SubjectPattern {
    return this.props.pattern
  }

  get messageType(): string {
    return this.props.messageType
  }

  get sourceId(): string {
    return this.props.sourceId
  }

  get createdAt(): Date {
    return this.props.createdAt
  }

  get updatedAt(): Date | null {
    return this.props.updatedAt
  }

  get health(): MappingHealth | undefined {
    return this.props.health
  }

  get healthDetail(): string | undefined {
    return this.props.healthDetail
  }

  matches(subject: string): boolean {
    return this.props.pattern.matches(subject)
  }

  looksLikeSpecificSubject(): boolean {
    return this.props.pattern.looksLikeSpecificSubject()
  }
}
