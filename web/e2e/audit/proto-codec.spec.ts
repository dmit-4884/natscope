import { test, expect } from '@playwright/test'
import * as A from './api-audit'
import { ns } from './fixtures-audit'
import { ensureAuditProtoSource, deleteAuditProtoSource, MESSAGE_TYPE } from './generators/protoSource'

/**
 * Slice 7 — Proto management + codec (the hard part): compile a non-trivial
 * proto (nested + enum + repeated + map), list types, encode/decode roundtrip,
 * validate JSON, and prove a published proto message decodes on list via a mapping.
 */
const N = ns('proto')
const SRC = 'AUDIT_CODEC_SRC'
let cid: string
let sourceId: string

test.beforeAll(async () => {
  cid = await A.getConnectionId('local')
  sourceId = await ensureAuditProtoSource(SRC)
  await A.deleteStreamIfExists(cid, N.stream)
})
test.afterAll(async () => {
  await A.deleteStreamIfExists(cid, N.stream)
  for (const m of (await A.listMappings()).mappings ?? []) {
    if (m.pattern.startsWith(N.mappingPrefix)) await A.deleteMapping(m.id).catch(() => {})
  }
  await deleteAuditProtoSource(SRC)
})

test.describe('proto: registry', () => {
  test('CHK compiled source lists its nested message types', async () => {
    const res = await A.listProtoMessages({ sourceId })
    const names = (res.messages ?? []).map((m) => m.fullName)
    expect(names).toContain('audit.codec.Outer')
    expect(names).toContain('audit.codec.Inner')
  })
})

test.describe('proto: codec roundtrip', () => {
  const json = '{"name":"hi","count":5,"tags":["a","b"],"color":"GREEN","inner":{"label":"L","depth":2},"scores":{"x":1}}'

  test('CHK EncodeMessage produces wire bytes; DecodeMessage recovers the values', async () => {
    const enc = (await A.encodeMessage({ messageType: MESSAGE_TYPE, data: json, sourceId })) as { result?: { data?: string; dataSize?: string } }
    expect(enc.result?.data, 'encoded bytes present').toBeTruthy()
    expect(Number(enc.result?.dataSize ?? '0')).toBeGreaterThan(0)

    const dec = (await A.decodeMessage({ data: enc.result!.data, messageType: MESSAGE_TYPE, sourceId })) as { result?: { data?: string } }
    const decoded = JSON.parse(dec.result?.data ?? '{}')
    expect(decoded.name).toBe('hi')
    expect(Number(decoded.count)).toBe(5)
    expect(decoded.color).toBe('GREEN')
    expect(decoded.inner?.label).toBe('L')
    expect(decoded.tags).toContain('b')
  })

  test('CHK ValidateJson accepts valid and rejects type-mismatched JSON', async () => {
    const ok = (await A.validateJson({ messageType: MESSAGE_TYPE, data: '{"name":"x","count":3}', sourceId })) as { result?: { valid?: boolean } }
    expect(ok.result?.valid).toBe(true)
    // Rejection surfaces either as result.valid=false (+error) or a Connect error.
    let rejected: boolean
    try {
      const bad = (await A.validateJson({ messageType: MESSAGE_TYPE, data: '{"count":"not-an-int"}', sourceId })) as { result?: { valid?: boolean; error?: string } }
      rejected = bad.result?.valid !== true && !!bad.result?.error
    } catch (e) {
      rejected = e instanceof A.ConnectError
    }
    expect(rejected).toBe(true)
  })
})

test.describe('proto: published message decodes on list via mapping', () => {
  test('CHK publish (server-encoded) then list decodes with the mapped type', async () => {
    await A.createStream(cid, N.stream, { subjects: [N.wildcard], storage: 1 })
    await A.createMapping({ pattern: N.wildcard, messageType: MESSAGE_TYPE, sourceId })

    const pub = await A.call<{ sequence?: string; error?: string }>(A.SVC.publish, 'PublishMessage', {
      connectionId: cid,
      subject: N.subject('evt'),
      subjectPattern: N.wildcard,
      messageType: MESSAGE_TYPE,
      sourceId,
      data: '{"name":"decoded-on-list","count":9,"color":"RED"}',
    })
    expect(pub.error ?? '').toBe('')

    const { messages } = await A.listMessages(cid, N.stream, { limit: '10', direction: 'DIRECTION_FORWARD' })
    const m = messages.find((x) => Number(x.sequence) === Number(pub.sequence))!
    expect(m.decodedType).toBe(MESSAGE_TYPE)
    expect(m.decodeError ?? '').toBe('')
  })
})
