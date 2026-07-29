/**
 * The single translation layer between server errors and the UI.
 *
 * The backend reports failures over two independent detail channels, and every
 * consumer in the app reads them through this module rather than poking at
 * `ConnectError.details` directly:
 *
 * - `google.rpc.ErrorInfo.reason` — domain failures (`NATS_STREAM_NOT_FOUND`,
 *   `CONNECTION_NAME_IN_USE`, …), attached by `internal/transports/grpc/helpers`.
 * - `io.altessa.badrequest.v1.BadRequest` — request-validation failures, one
 *   `FieldViolation` per broken rule, each carrying a canonical reason code
 *   produced by `internal/transports/grpc/reasoncodes`.
 */

import { ConnectError, Code } from '@connectrpc/connect'

import { ErrorInfoSchema } from '../gen/google/rpc/error_details_pb'
import { BadRequestSchema } from '../gen/io/altessa/badrequest/v1/badrequest_pb'

/** A single field-level validation failure reported by the server. */
export interface ValidationViolation {
  /** Dotted path of the offending field; empty for message-level (cross-field) rules. */
  fieldPath: string
  /** Canonical reason code from the backend catalog; empty when a rule maps to none. */
  code: string
  /** Server-supplied description of what the rule expected. */
  message: string
}

/**
 * Domain reason codes (`ErrorInfo.reason`) -> human-readable labels. Covers every
 * code the backend emits, from the central table in
 * `internal/transports/grpc/helpers/errors.go` and the per-handler `NewStatus`
 * calls. A code absent here falls back to the server's own message.
 */
const DOMAIN_REASON_LABELS: Record<string, string> = {
  // NATS
  NATS_PERMISSION_VIOLATION: 'Permission denied',
  NATS_TIMEOUT: 'Connection timed out',
  NATS_CONNECTION_FAILED: 'NATS server unavailable',
  NATS_CONNECTION_CLOSED: 'Connection closed',
  NATS_STREAM_NOT_FOUND: 'Stream not found',
  NATS_STREAM_NAME_IN_USE: 'Stream name already in use',
  NATS_CONSUMER_NOT_FOUND: 'Consumer not found',
  NATS_JETSTREAM_NOT_ENABLED: 'JetStream not enabled',
  NATS_MSG_NOT_FOUND: 'Message not found',
  NATS_MSG_DELETE_DENIED: 'Message deletion is disabled on this stream',
  NATS_WORKQUEUE_CONSUMER_NOT_ALLOWED: 'Cannot read a WorkQueue stream this way — messages would be consumed',
  NATS_BUCKET_NOT_FOUND: 'Bucket not found',
  NATS_BUCKET_EXISTS: 'Bucket already exists',
  NATS_KEY_NOT_FOUND: 'Key not found',
  NATS_NO_KEYS: 'No keys found',
  NATS_OBJECT_NOT_FOUND: 'Object not found',
  NATS_OBJECT_EXISTS: 'Object already exists',
  NATS_NO_OBJECTS: 'No objects found',
  // Connections
  CONNECTION_NOT_FOUND: 'Connection not found',
  CONNECTION_NAME_ALREADY_IN_USE: 'Connection name already in use',
  CONNECTION_URL_CREDENTIALS_MIXED:
    'Server URLs embed different credentials — use one set, or move them to the auth fields',
  CONNECTION_URL_CREDENTIALS_CONFLICT:
    'Credentials are set both in the server URL and in the auth fields — keep only one',
  // Proto
  PROTO_MESSAGE_NOT_FOUND: 'Proto message type not found',
  PROTO_DESCRIPTOR_NOT_FOUND: 'Proto descriptor not found',
  PROTO_SOURCE_NOT_FOUND: 'Proto source not found',
  PROTO_SOURCE_NAME_ALREADY_IN_USE: 'Proto source name already in use',
  PROTO_SELECTION_NOT_FOUND: 'Proto selection not found',
  PROTO_VERSION_NOT_FOUND: 'Proto version not found',
  PROTO_VERSION_ALREADY_EXISTS: 'Proto version already exists',
  NO_PROTO_SOURCES: 'No proto sources configured',
  MESSAGE_TYPE_NOT_IN_SOURCE: 'Message type is not in this source',
  SCHEMA_CONFLICT: 'Schema conflict',
  // Mappings
  MAPPING_NOT_FOUND: 'Mapping not found',
  MAPPING_PATTERN_ALREADY_IN_USE: 'Pattern is already mapped',
  MAPPING_SOURCE_NOT_FOUND: 'Mapping source not found',
  MAPPING_SOURCE_DISABLED: 'Mapping source is disabled',
  MAPPING_SOURCE_ID_REQUIRED: 'Mapping source is required',
  MAPPING_SELECTION_MISSING: 'Mapping selection is missing',
  MAPPING_DESCRIPTOR_MISSING: 'Mapping descriptor is missing',
  // Templates, settings, live, workspace
  MESSAGE_TEMPLATE_NOT_FOUND: 'Message template not found',
  SETTINGS_NOT_FOUND: 'Settings not found',
  LIVE_NO_SUBSCRIPTIONS: 'No live subscriptions could be created',
  WORKSPACE_INVALID_FILE: 'Invalid workspace file',
  WORKSPACE_UNKNOWN_SECTION: 'Unknown workspace section',
  WORKSPACE_SECTION_INVALID: 'Invalid workspace section payload',
  // Generic
  NOT_FOUND: 'Not found',
  DUPLICATE_ENTITY: 'Already exists',
  INVALID_REQUEST: 'Invalid request',
  INVALID_LIST_CURSOR: 'Invalid list cursor',
  PERMISSION_DENIED: 'Permission denied',
  UNAUTHORIZED: 'Not authenticated',
  DEADLINE_EXCEEDED: 'Request timed out',
  CANCELED: 'Request canceled',
  INTERNAL: 'Internal error',
}

/**
 * Validation reason codes (`FieldViolation.code`) -> labels. Mirrors the catalog
 * in `internal/transports/grpc/reasoncodes`; the required family is dynamic
 * (`<FIELD_NAME>_REQUIRED`) and is matched by suffix instead.
 */
const VALIDATION_REASON_LABELS: Record<string, string> = {
  INVALID_MIN_LENGTH_OR_VALUE: 'Below the allowed minimum',
  INVALID_MAX_LENGTH_OR_VALUE: 'Above the allowed maximum',
  INVALID_FORMAT_UUID: 'Must be a valid UUID',
  INVALID_FORMAT_EMAIL: 'Must be a valid email address',
  INVALID_FORMAT_REGEX: 'Does not match the required format',
  INVALID_FORMAT_URL: 'Must be a valid URL',
  INVALID_ENUM_VALUE: 'Not one of the allowed values',
  MUTUALLY_EXCLUSIVE_FIELDS: 'These fields cannot be used together',
  REQUIRED: 'This field is required',
}

/** Suffix the backend appends to a screaming-snake field name for required rules. */
const REQUIRED_SUFFIX = '_REQUIRED'

/** Used when a violation carries neither a message nor a known code. */
const GENERIC_VIOLATION_LABEL = 'Invalid value'

/** gRPC code -> human-readable label. */
const CODE_LABELS: Partial<Record<Code, string>> = {
  [Code.PermissionDenied]: 'Permission denied',
  [Code.NotFound]: 'Not found',
  [Code.AlreadyExists]: 'Already exists',
  [Code.Unavailable]: 'Service unavailable',
  [Code.DeadlineExceeded]: 'Request timed out',
  [Code.Unauthenticated]: 'Not authenticated',
  [Code.InvalidArgument]: 'Invalid request',
  [Code.Internal]: 'Internal error',
  [Code.ResourceExhausted]: 'Rate limited',
}

/** Semantic domain reason code from a ConnectError's ErrorInfo detail. */
export function getErrorReason(error: unknown): string | null {
  if (!(error instanceof ConnectError)) return null

  for (const info of error.findDetails(ErrorInfoSchema)) {
    if (info.reason) return info.reason
  }
  return null
}

/**
 * Every field violation from a ConnectError's BadRequest detail; empty for an
 * error that is not a validation failure.
 */
export function getValidationViolations(error: unknown): ValidationViolation[] {
  if (!(error instanceof ConnectError)) return []

  return error.findDetails(BadRequestSchema).flatMap(bad =>
    bad.fieldViolations.map(violation => ({
      fieldPath: violation.fieldPath ?? '',
      code: violation.code ?? '',
      message: violation.message ?? '',
    }))
  )
}

/** True when the server rejected the request with at least one field violation. */
export function isValidationError(error: unknown): boolean {
  return getValidationViolations(error).length > 0
}

/**
 * Violations keyed by field path, ready to bind to form fields. Message-level
 * rules carry no field path and are absent here — render those from
 * {@link getValidationViolations}. The first violation per field wins.
 */
export function getFieldErrors(error: unknown): Record<string, string> {
  const fieldErrors: Record<string, string> = {}

  for (const violation of getValidationViolations(error)) {
    if (violation.fieldPath && !(violation.fieldPath in fieldErrors)) {
      fieldErrors[violation.fieldPath] = describeViolationDetail(violation)
    }
  }
  return fieldErrors
}

/** Human-readable rendering of a single violation, field name included. */
export function describeViolation(violation: ValidationViolation): string {
  const field = humanizeFieldPath(violation.fieldPath)
  const detail = describeViolationDetail(violation)
  return field ? `${field}: ${detail}` : detail
}

/** Human-readable message from any error (ConnectError, Error, unknown). */
export function getErrorMessage(error: unknown): string {
  if (error instanceof ConnectError) {
    const reason = getErrorReason(error)
    if (reason && DOMAIN_REASON_LABELS[reason]) {
      return DOMAIN_REASON_LABELS[reason]
    }

    // The status message of a validation failure is always the bare
    // "Validation Failed"; the per-field detail is the only usable text.
    const violations = getValidationViolations(error)
    if (violations.length > 0) {
      return summarizeViolations(violations)
    }

    // Fall back to server message, stripping the [code] prefix
    const codePrefix = '[' + Code[error.code].toLowerCase() + '] '
    if (error.message && error.message !== '[' + Code[error.code].toLowerCase() + ']') {
      return error.message.startsWith(codePrefix) ? error.message.slice(codePrefix.length) : error.message
    }

    // Fall back to code label
    return CODE_LABELS[error.code] || `Error: ${Code[error.code]}`
  }

  if (error instanceof Error) {
    return error.message
  }

  return String(error)
}

export function stripErrorCodePrefix(message: string): string {
  return message.replace(/^\[\w+]\s*/, '')
}

/** True if error is a specific gRPC code. */
export function isErrorCode(error: unknown, code: Code): boolean {
  return error instanceof ConnectError && error.code === code
}

/** The detail half of a violation: what was wrong, without naming the field. */
function describeViolationDetail(violation: ValidationViolation): string {
  return violation.message || labelForValidationCode(violation.code) || GENERIC_VIOLATION_LABEL
}

/** Label for a validation reason code, including the dynamic required family. */
function labelForValidationCode(code: string): string {
  if (!code) return ''
  if (code in VALIDATION_REASON_LABELS) return VALIDATION_REASON_LABELS[code]
  if (code.endsWith(REQUIRED_SUFFIX)) return VALIDATION_REASON_LABELS.REQUIRED
  return ''
}

/** One line for a toast: the first violation, plus a count of the rest. */
function summarizeViolations(violations: ValidationViolation[]): string {
  const [first, ...rest] = violations
  const summary = describeViolation(first)
  return rest.length > 0 ? `${summary} (+${rest.length} more)` : summary
}

/** "items[2].start_seq" -> "Start seq"; empty for message-level rules. */
function humanizeFieldPath(fieldPath: string): string {
  const leaf = fieldPath.split('.').pop() ?? ''
  const words = leaf.replace(/\[\d+\]/g, '').split('_').filter(Boolean)
  if (words.length === 0) return ''

  const [first, ...rest] = words
  return [first.charAt(0).toUpperCase() + first.slice(1), ...rest].join(' ')
}
