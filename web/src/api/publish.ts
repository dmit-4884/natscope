import { publishClient } from './grpc/clients'

// Legacy interface types kept for existing consumers
export interface PublishRequest {
  connection_id: string
  subject: string
  subject_pattern?: string
  message_type?: string
  source_id?: string
  source_tag?: string
  data: Record<string, unknown>
  headers?: Record<string, string>
}

export interface PublishResponse {
  stream?: string
  sequence?: number
  duplicate?: boolean
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
  source_tag?: string
  data: Record<string, unknown>
}

export async function publishMessage(request: PublishRequest): Promise<PublishResponse> {
  const response = await publishClient.publishMessage({
    connectionId: request.connection_id,
    subject: request.subject,
    messageType: request.message_type || undefined,
    sourceId: request.source_id || undefined,
    sourceTag: request.source_tag || undefined,
    data: JSON.stringify(request.data),
    headers: request.headers ?? {},
    subjectPattern: request.subject_pattern || undefined,
  })
  if (response.error) {
    throw new Error(response.error)
  }
  return {
    stream: response.stream || undefined,
    sequence: Number(response.sequence) || undefined,
    duplicate: response.duplicate || false,
  }
}

export async function validateJSON(request: ValidateJSONRequest): Promise<ValidationResult> {
  const response = await publishClient.validateJson({
    messageType: request.message_type,
    sourceId: request.source_id,
    sourceTag: request.source_tag || undefined,
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
