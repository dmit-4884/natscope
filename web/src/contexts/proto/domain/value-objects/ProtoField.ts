import { ValueObject } from '@/shared'

export type FieldLabel = 'optional' | 'required' | 'repeated'

interface ProtoFieldProps {
  name: string
  type: string
  number: number
  label: FieldLabel
  isMessage: boolean
}

/** A field in a protobuf message definition. */
export class ProtoField extends ValueObject<ProtoFieldProps> {
  static fromApi(data: {
    name: string
    type: string
    number: number
    label?: string
    is_message?: boolean
  }): ProtoField {
    return new ProtoField({
      name: data.name,
      type: data.type,
      number: data.number,
      label: (data.label ?? 'optional') as FieldLabel,
      isMessage: data.is_message ?? false,
    })
  }

  get name(): string {
    return this.props.name
  }

  get type(): string {
    return this.props.type
  }

  get number(): number {
    return this.props.number
  }

  get label(): FieldLabel {
    return this.props.label
  }

  get isMessage(): boolean {
    return this.props.isMessage
  }

  isRepeated(): boolean {
    return this.props.label === 'repeated'
  }

  isRequired(): boolean {
    return this.props.label === 'required'
  }

  isScalar(): boolean {
    const scalarTypes = [
      'double', 'float', 'int32', 'int64', 'uint32', 'uint64',
      'sint32', 'sint64', 'fixed32', 'fixed64', 'sfixed32', 'sfixed64',
      'bool', 'string', 'bytes',
    ]
    return scalarTypes.includes(this.props.type)
  }

  typeDisplayName(): string {
    if (this.isRepeated()) {
      return `repeated ${this.props.type}`
    }
    return this.props.type
  }
}
