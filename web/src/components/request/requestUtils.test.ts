import { describe, expect, it } from 'vitest'
import { Code, ConnectError } from '@connectrpc/connect'
import { create, toBinary } from '@bufbuild/protobuf'
import { ErrorInfoSchema } from '@/gen/google/rpc/error_details_pb'
import {
  formatTimeout,
  guessReplyType,
  literalSubjectError,
  literalSubjects,
  requestFailureKind,
} from './requestUtils'

function connectError(code: Code, reason: string): ConnectError {
  const err = new ConnectError('failed', code)
  err.details = [{ type: ErrorInfoSchema.typeName, value: toBinary(ErrorInfoSchema, create(ErrorInfoSchema, { reason })) }]
  return err
}

describe('literalSubjectError', () => {
  it.each(['svc.echo', '$SRV.PING', '$JS.API.INFO', 'orders.{{uuid}}', ''])('accepts %j', (subject) => {
    expect(literalSubjectError(subject)).toBeNull()
  })

  it.each([
    ['svc.*', 'wildcards'],
    ['svc.>', 'wildcards'],
    ['svc..echo', 'empty tokens'],
    ['svc.', 'empty tokens'],
    ['svc echo', 'spaces'],
  ])('rejects %j', (subject, fragment) => {
    expect(literalSubjectError(subject)).toContain(fragment)
  })
})

describe('literalSubjects', () => {
  it('keeps only literal patterns', () => {
    expect(literalSubjects(['orders.*', 'svc.echo', 'events.>', '$SRV.PING'])).toEqual(['svc.echo', '$SRV.PING'])
  })
})

describe('requestFailureKind', () => {
  it('classifies by the server reason code', () => {
    expect(requestFailureKind(connectError(Code.Unavailable, 'NATS_NO_RESPONDERS'))).toBe('no-responders')
    expect(requestFailureKind(connectError(Code.DeadlineExceeded, 'NATS_TIMEOUT'))).toBe('timeout')
    expect(requestFailureKind(connectError(Code.DeadlineExceeded, 'DEADLINE_EXCEEDED'))).toBe('timeout')
    expect(requestFailureKind(connectError(Code.PermissionDenied, 'NATS_PERMISSION_VIOLATION'))).toBe('error')
    expect(requestFailureKind(new Error('boom'))).toBe('error')
  })
})

describe('formatTimeout', () => {
  it('uses ms below a second and seconds above', () => {
    expect(formatTimeout(500)).toBe('500 ms')
    expect(formatTimeout(5000)).toBe('5 s')
  })
})

describe('guessReplyType', () => {
  const candidates = [
    { id: 'src|svc.v1.PingResponse', fullName: 'svc.v1.PingResponse', sourceId: 'src' },
    { id: 'other|svc.v1.EchoReply', fullName: 'svc.v1.EchoReply', sourceId: 'other' },
    { id: 'src|svc.v1.EchoReply', fullName: 'svc.v1.EchoReply', sourceId: 'src' },
  ]

  it('pairs a Request type with its Response or Reply in the same source', () => {
    expect(guessReplyType('svc.v1.PingRequest', 'src', candidates)).toBe('src|svc.v1.PingResponse')
    expect(guessReplyType('svc.v1.EchoRequest', 'src', candidates)).toBe('src|svc.v1.EchoReply')
  })

  it('returns nothing without a matching pair', () => {
    expect(guessReplyType('svc.v1.Ping', 'src', candidates)).toBe('')
    expect(guessReplyType('svc.v1.MissingRequest', 'src', candidates)).toBe('')
    expect(guessReplyType(undefined, undefined, candidates)).toBe('')
  })
})
