import { Entity } from '@/shared'
import { ProtoField } from '../value-objects/ProtoField'

interface ProtoMessageProps {
  id: string
  fullName: string
  protoFile: string
  packageName: string
  fields: ProtoField[]
  /** ProtoSource the message lives in. Required after the registry redesign. */
  sourceId: string
  /** Active tag of the source snapshot ("local" / "v1.0.0" / etc.). */
  sourceTag: string
}

/**
 * Proto message type within a source snapshot. Identity = (sourceId, fullName):
 * same FQN in different sources are NOT equal (may declare different schemas).
 */
export class ProtoMessage extends Entity<ProtoMessageProps> {
  private constructor(props: ProtoMessageProps) {
    super(props)
  }

  static fromApi(data: {
    full_name: string
    proto_file: string
    package: string
    source_id: string
    source_tag: string
    fields: Array<{
      name: string
      type: string
      number: number
      label?: string
      is_message?: boolean
    }>
  }): ProtoMessage {
    return new ProtoMessage({
      id: `${data.source_id}|${data.full_name}`,
      fullName: data.full_name,
      protoFile: data.proto_file,
      packageName: data.package,
      sourceId: data.source_id,
      sourceTag: data.source_tag,
      fields: data.fields.map((f) => ProtoField.fromApi(f)),
    })
  }

  get fullName(): string {
    return this.props.fullName
  }

  get protoFile(): string {
    return this.props.protoFile
  }

  get packageName(): string {
    return this.props.packageName
  }

  get sourceId(): string {
    return this.props.sourceId
  }

  get sourceTag(): string {
    return this.props.sourceTag
  }

  get fields(): ReadonlyArray<ProtoField> {
    return this.props.fields
  }

  get fieldCount(): number {
    return this.props.fields.length
  }

  /** Returns just the message name without package. */
  shortName(): string {
    const parts = this.props.fullName.split('.')
    return parts[parts.length - 1]
  }
}
