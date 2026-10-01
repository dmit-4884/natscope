import { SchemaTypeKind as PbKind } from '../gen/types/proto/schema_type_pb'
import type {
  SchemaType as PbType,
  SchemaField as PbField,
  SchemaMessage as PbMessage,
  SchemaEnum as PbEnum,
  SchemaService as PbService,
} from '../gen/types/proto/schema_type_pb'
import { registryClient } from './grpc/clients'

export type SchemaTypeKind = 'message' | 'enum' | 'service'

export interface SchemaType {
  id: string
  fullName: string
  kind: SchemaTypeKind
  file: string
  packageName: string
  comment: string
  memberCount: number
  dependency: boolean
  sourceId: string
  sourceRevision: string
}

export interface SchemaField {
  name: string
  jsonName: string
  number: number
  kind: string
  typeName: string
  repeated: boolean
  mapKey: string
  optional: boolean
  required: boolean
  oneof: string
  deprecated: boolean
  comment: string
}

export interface SchemaMessage {
  fullName: string
  file: string
  comment: string
  deprecated: boolean
  fields: SchemaField[]
}

export interface SchemaEnumValue {
  name: string
  number: number
  comment: string
  deprecated: boolean
}

export interface SchemaEnum {
  fullName: string
  file: string
  comment: string
  deprecated: boolean
  values: SchemaEnumValue[]
}

export interface SchemaMethod {
  name: string
  inputType: string
  outputType: string
  clientStreaming: boolean
  serverStreaming: boolean
  comment: string
  deprecated: boolean
}

export interface SchemaService {
  fullName: string
  file: string
  comment: string
  deprecated: boolean
  methods: SchemaMethod[]
}

export interface TypeDescription {
  messages: SchemaMessage[]
  enums: SchemaEnum[]
  services: SchemaService[]
}

export interface ProtoExampleResponse {
  message_type: string
  example: Record<string, unknown>
}

const KIND_NAMES: Record<PbKind, SchemaTypeKind> = {
  [PbKind.UNSPECIFIED]: 'message',
  [PbKind.MESSAGE]: 'message',
  [PbKind.ENUM]: 'enum',
  [PbKind.SERVICE]: 'service',
}

function schemaTypeId(sourceId: string, fullName: string): string {
  return `${sourceId}|${fullName}`
}

function toType(t: PbType): SchemaType {
  return {
    id: schemaTypeId(t.sourceId, t.fullName),
    fullName: t.fullName,
    kind: KIND_NAMES[t.kind],
    file: t.file,
    packageName: t.package,
    comment: t.comment,
    memberCount: t.memberCount,
    dependency: t.dependency,
    sourceId: t.sourceId,
    sourceRevision: t.sourceRevision,
  }
}

function toField(f: PbField): SchemaField {
  return {
    name: f.name,
    jsonName: f.jsonName,
    number: f.number,
    kind: f.kind,
    typeName: f.typeName,
    repeated: f.repeated,
    mapKey: f.mapKey,
    optional: f.optional,
    required: f.required,
    oneof: f.oneof,
    deprecated: f.deprecated,
    comment: f.comment,
  }
}

function toMessage(m: PbMessage): SchemaMessage {
  return { fullName: m.fullName, file: m.file, comment: m.comment, deprecated: m.deprecated, fields: m.fields.map(toField) }
}

function toEnum(e: PbEnum): SchemaEnum {
  return {
    fullName: e.fullName,
    file: e.file,
    comment: e.comment,
    deprecated: e.deprecated,
    values: e.values.map((v) => ({ name: v.name, number: v.number, comment: v.comment, deprecated: v.deprecated })),
  }
}

function toService(s: PbService): SchemaService {
  return {
    fullName: s.fullName,
    file: s.file,
    comment: s.comment,
    deprecated: s.deprecated,
    methods: s.methods.map((m) => ({
      name: m.name,
      inputType: m.inputType,
      outputType: m.outputType,
      clientStreaming: m.clientStreaming,
      serverStreaming: m.serverStreaming,
      comment: m.comment,
      deprecated: m.deprecated,
    })),
  }
}

export async function listSchemaTypes(sourceId?: string): Promise<SchemaType[]> {
  const response = await registryClient.listTypes({ sourceId })
  return response.types.map(toType)
}

export async function describeSchemaType(
  sourceId: string,
  fullName: string,
  includeReachable = false,
): Promise<TypeDescription> {
  const response = await registryClient.describeType({ sourceId, fullName, includeReachable })
  return {
    messages: response.messages.map(toMessage),
    enums: response.enums.map(toEnum),
    services: response.services.map(toService),
  }
}

export async function getProtoMessageExample(
  sourceId: string,
  messageName: string,
): Promise<ProtoExampleResponse> {
  if (!sourceId) {
    throw new Error('getProtoMessageExample: sourceId is required')
  }
  const response = await registryClient.generateExample({ fullName: messageName, sourceId })
  return {
    message_type: messageName,
    example: JSON.parse(response.json || '{}'),
  }
}
