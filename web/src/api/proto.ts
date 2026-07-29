import { registryClient } from './grpc/clients'

// Source-aware shape: each ProtoMessage is annotated with the source it lives in.
export interface ProtoMessage {
  full_name: string
  proto_file: string
  package: string
  fields: ProtoField[]
  source_id: string
  source_tag: string
}

interface ProtoField {
  name: string
  type: string
  number: number
  label?: string
  is_message?: boolean
}

export interface ProtoMessagesResponse {
  messages: ProtoMessage[]
  /** Keyed by `${sourceId}|${package}` so same FQN across sources never collides. */
  grouped_by_source_package: Record<string, string[]>
}

export interface ProtoExampleResponse {
  message_type: string
  example: Record<string, unknown>
}

export async function getProtoMessages(): Promise<ProtoMessagesResponse> {
  const response = await registryClient.listProtoMessages({})
  const messages: ProtoMessage[] = response.messages.map((m) => ({
    full_name: m.fullName,
    proto_file: m.protoFile,
    package: m.package,
    source_id: m.sourceId,
    source_tag: m.sourceTag,
    fields: m.fields.map((f) => ({
      name: f.name,
      type: f.type,
      number: f.number,
      label: f.label || undefined,
      is_message: f.isMessage || undefined,
    })),
  }))

  const grouped_by_source_package: Record<string, string[]> = {}
  for (const msg of messages) {
    const key = `${msg.source_id}|${msg.package}`
    if (!grouped_by_source_package[key]) {
      grouped_by_source_package[key] = []
    }
    grouped_by_source_package[key].push(msg.full_name)
  }
  return { messages, grouped_by_source_package }
}

export async function getProtoMessage(sourceId: string, messageName: string): Promise<ProtoMessage> {
  if (!sourceId) {
    throw new Error('getProtoMessage: sourceId is required')
  }
  const response = await registryClient.getProtoMessage({ fullName: messageName, sourceId })
  const m = response.message!
  return {
    full_name: m.fullName,
    proto_file: m.protoFile,
    package: m.package,
    source_id: m.sourceId,
    source_tag: m.sourceTag,
    fields: m.fields.map((f) => ({
      name: f.name,
      type: f.type,
      number: f.number,
      label: f.label || undefined,
      is_message: f.isMessage || undefined,
    })),
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
