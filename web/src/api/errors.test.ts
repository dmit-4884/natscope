import { describe, it, expect } from 'vitest'
import { Code, ConnectError } from '@connectrpc/connect'
import { create, toBinary, type DescMessage, type MessageInitShape } from '@bufbuild/protobuf'

import { ErrorInfoSchema } from '../gen/google/rpc/error_details_pb'
import { BadRequestSchema } from '../gen/io/altessa/badrequest/v1/badrequest_pb'
import {
  describeViolation,
  getErrorMessage,
  getErrorReason,
  getFieldErrors,
  getValidationViolations,
  isErrorCode,
  isValidationError,
} from './errors'

type FieldViolationInit = NonNullable<MessageInitShape<typeof BadRequestSchema>['fieldViolations']>[number]

/**
 * Build the error a real transport hands us: details arrive as binary
 * IncomingDetails (type name + serialized bytes), not decoded messages.
 */
function wireError<T extends DescMessage>(code: Code, message: string, schema: T, init: MessageInitShape<T>): ConnectError {
  const err = new ConnectError(message, code)
  err.details = [{ type: schema.typeName, value: toBinary(schema, create(schema, init)) }]
  return err
}

function validationError(...fieldViolations: FieldViolationInit[]): ConnectError {
  return wireError(Code.InvalidArgument, 'Validation Failed', BadRequestSchema, { fieldViolations })
}

function domainError(code: Code, message: string, reason: string): ConnectError {
  return wireError(code, message, ErrorInfoSchema, { reason })
}

function natsAPIError(message: string, errCode: string): ConnectError {
  return wireError(Code.InvalidArgument, message, ErrorInfoSchema, {
    reason: 'NATS_API_ERROR',
    domain: 'nats.jetstream',
    metadata: { err_code: errCode, http_code: '400' },
  })
}

describe('domain reason labels', () => {
  // Guards against the table drifting from the backend, which is what let
  // CONNECTION_NAME_IN_USE sit here dead while the server sent
  // CONNECTION_NAME_ALREADY_IN_USE. Keep in sync with the codes emitted by
  // internal/transports/grpc/helpers/errors.go and the handlers' NewStatus calls.
  const BACKEND_REASON_CODES = [
    'NATS_PERMISSION_VIOLATION', 'NATS_TIMEOUT', 'NATS_CONNECTION_FAILED', 'NATS_CONNECTION_CLOSED',
    'NATS_STREAM_NOT_FOUND', 'NATS_STREAM_NAME_IN_USE', 'NATS_CONSUMER_NOT_FOUND', 'NATS_JETSTREAM_NOT_ENABLED',
    'NATS_MSG_NOT_FOUND', 'NATS_MSG_DELETE_DENIED', 'NATS_WORKQUEUE_CONSUMER_NOT_ALLOWED',
    'NATS_BUCKET_NOT_FOUND', 'NATS_BUCKET_EXISTS',
    'NATS_KEY_NOT_FOUND', 'NATS_NO_KEYS', 'NATS_OBJECT_NOT_FOUND', 'NATS_OBJECT_EXISTS', 'NATS_NO_OBJECTS',
    'CONNECTION_NOT_FOUND', 'CONNECTION_NAME_ALREADY_IN_USE',
    'PROTO_MESSAGE_NOT_FOUND', 'PROTO_DESCRIPTOR_NOT_FOUND', 'PROTO_SOURCE_NOT_FOUND',
    'PROTO_SOURCE_NAME_ALREADY_IN_USE', 'PROTO_SELECTION_NOT_FOUND', 'PROTO_VERSION_NOT_FOUND',
    'PROTO_VERSION_ALREADY_EXISTS', 'NO_PROTO_SOURCES', 'MESSAGE_TYPE_NOT_IN_SOURCE', 'SCHEMA_CONFLICT',
    'MAPPING_NOT_FOUND', 'MAPPING_PATTERN_ALREADY_IN_USE', 'MAPPING_SOURCE_NOT_FOUND',
    'MAPPING_SOURCE_DISABLED', 'MAPPING_SOURCE_ID_REQUIRED', 'MAPPING_SELECTION_MISSING',
    'MAPPING_DESCRIPTOR_MISSING',
    'MESSAGE_TEMPLATE_NOT_FOUND', 'SETTINGS_NOT_FOUND', 'LIVE_NO_SUBSCRIPTIONS',
    'WORKSPACE_INVALID_FILE', 'WORKSPACE_UNKNOWN_SECTION', 'WORKSPACE_SECTION_INVALID',
    'NOT_FOUND', 'DUPLICATE_ENTITY', 'INVALID_REQUEST', 'INVALID_LIST_CURSOR', 'PERMISSION_DENIED',
    'UNAUTHORIZED', 'DEADLINE_EXCEEDED', 'CANCELED', 'INTERNAL',
  ]

  it.each(BACKEND_REASON_CODES)('labels %s', code => {
    const rendered = getErrorMessage(domainError(Code.Unknown, 'raw server text', code))
    expect(rendered).not.toBe('raw server text')
    expect(rendered).not.toBe('')
  })
})

describe('getErrorReason', () => {
  it('reads the reason off an ErrorInfo detail', () => {
    expect(getErrorReason(domainError(Code.NotFound, 'stream missing', 'NATS_STREAM_NOT_FOUND'))).toBe('NATS_STREAM_NOT_FOUND')
  })

  it('returns null when there is no ErrorInfo', () => {
    expect(getErrorReason(new ConnectError('boom', Code.Internal))).toBeNull()
    expect(getErrorReason(new Error('boom'))).toBeNull()
  })
})

describe('getValidationViolations', () => {
  it('decodes every field violation from the BadRequest detail', () => {
    const err = validationError(
      { fieldPath: 'name', code: 'NAME_REQUIRED', message: 'value is required' },
      { fieldPath: 'urls', code: 'INVALID_MIN_LENGTH_OR_VALUE', message: 'value must contain at least 1 item' }
    )

    expect(getValidationViolations(err)).toEqual([
      { fieldPath: 'name', code: 'NAME_REQUIRED', message: 'value is required' },
      { fieldPath: 'urls', code: 'INVALID_MIN_LENGTH_OR_VALUE', message: 'value must contain at least 1 item' },
    ])
  })

  it('defaults absent optional fields to empty strings', () => {
    const err = validationError({ code: 'MUTUALLY_EXCLUSIVE_FIELDS' })
    expect(getValidationViolations(err)).toEqual([{ fieldPath: '', code: 'MUTUALLY_EXCLUSIVE_FIELDS', message: '' }])
  })

  it('is empty for non-validation errors', () => {
    expect(getValidationViolations(new ConnectError('boom', Code.Internal))).toEqual([])
    expect(getValidationViolations('boom')).toEqual([])
  })
})

describe('isValidationError', () => {
  it('distinguishes field violations from other failures', () => {
    expect(isValidationError(validationError({ fieldPath: 'name', code: 'NAME_REQUIRED' }))).toBe(true)
    expect(isValidationError(new ConnectError('nope', Code.InvalidArgument))).toBe(false)
  })
})

describe('describeViolation', () => {
  // Every code the backend catalog (internal/transports/grpc/reasoncodes) can emit.
  const cases: Array<[string, string, string]> = [
    ['NAME_REQUIRED', 'name', 'Name: This field is required'],
    ['CONNECTION_ID_REQUIRED', 'connection_id', 'Connection id: This field is required'],
    ['REQUIRED', 'name', 'Name: This field is required'],
    ['INVALID_MIN_LENGTH_OR_VALUE', 'max_msgs', 'Max msgs: Below the allowed minimum'],
    ['INVALID_MAX_LENGTH_OR_VALUE', 'max_msgs', 'Max msgs: Above the allowed maximum'],
    ['INVALID_FORMAT_UUID', 'id', 'Id: Must be a valid UUID'],
    ['INVALID_FORMAT_EMAIL', 'email', 'Email: Must be a valid email address'],
    ['INVALID_FORMAT_REGEX', 'subject', 'Subject: Does not match the required format'],
    ['INVALID_FORMAT_URL', 'url', 'Url: Must be a valid URL'],
    ['INVALID_ENUM_VALUE', 'storage', 'Storage: Not one of the allowed values'],
    ['MUTUALLY_EXCLUSIVE_FIELDS', '', 'These fields cannot be used together'],
  ]

  it.each(cases)('labels %s without a server message', (code, fieldPath, expected) => {
    expect(describeViolation({ code, fieldPath, message: '' })).toBe(expected)
  })

  it('prefers the server message over the code label', () => {
    const rendered = describeViolation({
      fieldPath: 'urls',
      code: 'INVALID_MIN_LENGTH_OR_VALUE',
      message: 'value must contain at least 1 item',
    })
    expect(rendered).toBe('Urls: value must contain at least 1 item')
  })

  it('falls back when the code is unknown and there is no message', () => {
    expect(describeViolation({ fieldPath: 'name', code: 'SOMETHING_NEW', message: '' })).toBe('Name: Invalid value')
  })

  it('names the leaf of a nested or repeated path', () => {
    expect(describeViolation({ fieldPath: 'config.sources[2].start_seq', code: '', message: 'bad' })).toBe('Start seq: bad')
  })
})

describe('getFieldErrors', () => {
  it('keys violations by field path for form binding', () => {
    const err = validationError(
      { fieldPath: 'name', code: 'NAME_REQUIRED', message: 'value is required' },
      { fieldPath: 'urls', code: 'INVALID_MIN_LENGTH_OR_VALUE' }
    )

    expect(getFieldErrors(err)).toEqual({
      name: 'value is required',
      urls: 'Below the allowed minimum',
    })
  })

  it('keeps the first violation per field and drops message-level rules', () => {
    const err = validationError(
      { fieldPath: 'name', message: 'first' },
      { fieldPath: 'name', message: 'second' },
      { code: 'MUTUALLY_EXCLUSIVE_FIELDS' }
    )
    expect(getFieldErrors(err)).toEqual({ name: 'first' })
  })
})

describe('getErrorMessage', () => {
  it('prefers the domain reason label', () => {
    expect(getErrorMessage(domainError(Code.NotFound, 'whatever', 'NATS_STREAM_NOT_FOUND'))).toBe('Stream not found')
  })

  it('renders a lost compare-and-swap as a reload prompt', () => {
    expect(getErrorMessage(natsAPIError('wrong last sequence: 7', '10071')))
      .toBe('The value changed since you loaded it — reload it and reapply your change')
  })

  it('keeps the server text for an unmapped JetStream error code', () => {
    expect(getErrorMessage(natsAPIError('maximum consumers limit reached', '10026')))
      .toBe('maximum consumers limit reached')
  })

  it('renders a single violation instead of the bare "Validation Failed"', () => {
    const err = validationError({ fieldPath: 'name', code: 'NAME_REQUIRED' })
    expect(getErrorMessage(err)).toBe('Name: This field is required')
  })

  it('renders a message-level rule with no field prefix', () => {
    const err = validationError({
      code: 'MUTUALLY_EXCLUSIVE_FIELDS',
      message: 'start_seq and start_time are mutually exclusive',
    })
    expect(getErrorMessage(err)).toBe('start_seq and start_time are mutually exclusive')
  })

  it('counts the remaining violations', () => {
    const err = validationError(
      { fieldPath: 'name', code: 'NAME_REQUIRED' },
      { fieldPath: 'urls', code: 'INVALID_MIN_LENGTH_OR_VALUE' },
      { fieldPath: 'id', code: 'INVALID_FORMAT_UUID' }
    )
    expect(getErrorMessage(err)).toBe('Name: This field is required (+2 more)')
  })

  it('falls back to the server message, stripping the code prefix', () => {
    expect(getErrorMessage(new ConnectError('something broke', Code.Internal))).toBe('something broke')
  })

  it.each([
    [Code.InvalidArgument, 'connection name is required'],
    [Code.NotFound, 'consumer not found'],
    [Code.AlreadyExists, 'stream name already in use'],
    [Code.PermissionDenied, 'not allowed'],
    [Code.DeadlineExceeded, 'took too long'],
    [Code.FailedPrecondition, 'stream is sealed'],
    [Code.ResourceExhausted, 'too many requests'],
    [Code.OutOfRange, 'sequence past the end'],
  ])('strips the multi-word %s code prefix', (code, text) => {
    expect(getErrorMessage(new ConnectError(text, code))).toBe(text)
  })

  it('falls back to a code label when there is no message', () => {
    expect(getErrorMessage(new ConnectError('', Code.Unavailable))).toBe('Service unavailable')
  })

  it('falls back to a code label when the message is only a multi-word code prefix', () => {
    expect(getErrorMessage(new ConnectError('', Code.InvalidArgument))).toBe('Invalid request')
  })

  it('handles plain errors and unknown values', () => {
    expect(getErrorMessage(new Error('plain'))).toBe('plain')
    expect(getErrorMessage('just a string')).toBe('just a string')
  })
})

describe('isErrorCode', () => {
  it('matches only ConnectErrors with the given code', () => {
    expect(isErrorCode(new ConnectError('x', Code.NotFound), Code.NotFound)).toBe(true)
    expect(isErrorCode(new ConnectError('x', Code.NotFound), Code.Internal)).toBe(false)
    expect(isErrorCode(new Error('x'), Code.NotFound)).toBe(false)
  })
})
