import { test, expect } from '@playwright/test'
import * as A from './api-audit'
import { ns } from './fixtures-audit'

/**
 * Slice 14 — Transport & domain-error → Connect-code mapping (API-level).
 * Asserts the codes handlers return for the canonical error paths.
 */
const N = ns('errors')
let cid: string

async function codeOf(p: Promise<unknown>): Promise<string> {
  try {
    await p
    return 'NO_ERROR'
  } catch (e) {
    if (e instanceof A.ConnectError) return e.code
    throw e
  }
}

test.beforeAll(async () => {
  cid = await A.getConnectionId('local')
  await A.deleteStreamIfExists(cid, N.stream)
})
test.afterAll(async () => {
  await A.deleteStreamIfExists(cid, N.stream)
})

test.describe('error-code mapping', () => {
  test('CHK invalid_argument for buf.validate failure (empty required field)', async () => {
    expect(await codeOf(A.createStream(cid, '', { subjects: ['x.>'] }))).toBe('invalid_argument')
  })

  test('CHK not_found for a missing stream', async () => {
    expect(await codeOf(A.getStream(cid, 'AUDIT_ERRORS_NOPE_XYZ'))).toBe('not_found')
  })

  test('CHK not_found for a missing message sequence', async () => {
    await A.createStream(cid, N.stream, { subjects: [N.wildcard], storage: 1 })
    expect(await codeOf(A.getMessage(cid, N.stream, 999999))).toBe('not_found')
  })

  test('CHK failed_precondition when deleting a message on a deny_delete stream', async () => {
    const s = `${N.stream}_DENY`
    await A.deleteStreamIfExists(cid, s)
    await A.createStream(cid, s, { subjects: [`${N.subjectRoot}d.>`], storage: 1, denyDelete: true })
    const pub = await A.publish(cid, `${N.subjectRoot}d.x`, `${N.subjectRoot}d.>`, 'x')
    expect(await codeOf(A.deleteMessage(cid, s, Number(pub.sequence ?? '1')))).toBe('failed_precondition')
    await A.deleteStreamIfExists(cid, s)
  })

  test('CHK invalid_argument for a non-UUID connection id on DeleteConnection', async () => {
    expect(await codeOf(A.deleteConnection('not-a-uuid'))).toBe('invalid_argument')
  })

  test('CHK stream op against a bogus connection id errors (not silently ok)', async () => {
    const code = await codeOf(A.listStreams('00000000-0000-0000-0000-000000000000'))
    expect(code).not.toBe('NO_ERROR')
  })

  test('CHK unknown RPC path is rejected by the transport', async () => {
    const code = await codeOf(A.call(A.SVC.streams, 'NoSuchMethod', {}))
    expect(['unimplemented', 'not_found', '404', '405']).toContain(code)
  })
})
