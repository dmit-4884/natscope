import { encodeBytesToBase64 } from '@/utils/base64'
import { durToMillis, millisToDur } from '@/utils/timestamp'
import { framingToProto, type Framing } from './framing'
import { publishClient } from './grpc/clients'

// Legacy interface types kept for existing consumers
export interface PublishRequest {
  connection_id: string
  subject: string
  subject_pattern?: string
  message_type?: string
  source_id?: string
  schema_fingerprint?: string
  framing?: Framing
  /** null publishes without a body (counter increments). */
  data: Record<string, unknown> | null
  headers?: Record<string, string>
}

export interface PublishResponse {
  stream?: string
  sequence?: number
  duplicate?: boolean
  counter_value?: string
}

export interface ValidationResult {
  valid: boolean
  violations?: ValidationViolation[]
  error?: string
}

interface ValidationViolation {
  field_path: string
  message: string
  rule_id?: string
}

export interface ValidateJSONRequest {
  message_type: string
  source_id: string
  schema_fingerprint?: string
  data: Record<string, unknown>
}

export async function publishMessage(request: PublishRequest): Promise<PublishResponse> {
  const response = await publishClient.publishMessage({
    connectionId: request.connection_id,
    subject: request.subject,
    messageType: request.message_type || undefined,
    sourceId: request.source_id || undefined,
    schemaFingerprint: request.schema_fingerprint || undefined,
    data: request.data === null ? '' : JSON.stringify(request.data),
    headers: request.headers ?? {},
    subjectPattern: request.subject_pattern || undefined,
    framing: framingToProto(request.framing),
  })
  if (response.error) {
    throw new Error(response.error)
  }
  return {
    stream: response.stream || undefined,
    sequence: Number(response.sequence) || undefined,
    duplicate: response.duplicate || false,
    counter_value: response.counterValue,
  }
}

export interface CorePublishRequest {
  connection_id: string
  subject: string
  data: string
  headers?: Record<string, string>
  message_type?: string
  source_id?: string
  schema_fingerprint?: string
  framing?: Framing
}

export async function publishCoreMessage(request: CorePublishRequest): Promise<void> {
  const response = await publishClient.publishMessage({
    connectionId: request.connection_id,
    subject: request.subject,
    data: request.data,
    headers: request.headers ?? {},
    messageType: request.message_type || undefined,
    sourceId: request.source_id || undefined,
    schemaFingerprint: request.schema_fingerprint || undefined,
    framing: framingToProto(request.framing),
    core: true,
  })
  if (response.error) {
    throw new Error(response.error)
  }
}

export interface RequestMessageRequest {
  connection_id: string
  subject: string
  data: string
  headers?: Record<string, string>
  message_type?: string
  source_id?: string
  schema_fingerprint?: string
  framing?: Framing
  timeout_ms?: number
}

export interface RequestReply {
  subject: string
  data_base64: string
  size: number
  headers: Record<string, string>
  duration_ms: number
}

export async function requestMessage(request: RequestMessageRequest): Promise<RequestReply> {
  const response = await publishClient.requestMessage({
    connectionId: request.connection_id,
    subject: request.subject,
    data: request.data,
    headers: request.headers ?? {},
    messageType: request.message_type || undefined,
    sourceId: request.source_id || undefined,
    schemaFingerprint: request.schema_fingerprint || undefined,
    framing: framingToProto(request.framing),
    timeout: millisToDur(request.timeout_ms),
  })
  return {
    subject: response.subject,
    data_base64: encodeBytesToBase64(response.data),
    size: response.data.length,
    headers: response.headers,
    duration_ms: durToMillis(response.duration) ?? 0,
  }
}

export async function validateJSON(request: ValidateJSONRequest): Promise<ValidationResult> {
  const response = await publishClient.validateJson({
    messageType: request.message_type,
    sourceId: request.source_id,
    schemaFingerprint: request.schema_fingerprint || undefined,
    data: JSON.stringify(request.data),
  })
  const result = response.result
  return {
    valid: result?.valid ?? false,
    violations: result?.violations?.map(v => ({
      field_path: v.fieldPath,
      message: v.message,
      rule_id: v.constraintId || undefined,
    })),
    error: result?.error,
  }
}
