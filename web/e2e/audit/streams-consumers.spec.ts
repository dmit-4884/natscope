import { test, expect } from '@playwright/test'
import * as A from './api-audit'
import { ns } from './fixtures-audit'

/**
 * Slice 2 — Streams & consumers (API-level lifecycle, validation bounds,
 * and suspected-bug probes surfaced by discovery). workers:1, self-cleaning.
 */
const N = ns('streams')
let cid: string

test.beforeAll(async () => {
  cid = await A.getConnectionId('local')
  await A.deleteStreamIfExists(cid, N.stream)
  await A.deleteStreamIfExists(cid, N.stream2)
})
test.afterAll(async () => {
  await A.deleteStreamIfExists(cid, N.stream)
  await A.deleteStreamIfExists(cid, N.stream2)
})

test.describe('streams: lifecycle', () => {
  test('CHK create limits/file stream and read back config', async () => {
    const res = await A.createStream(cid, N.stream, {
      subjects: [N.wildcard],
      retention: 0,
      storage: 0,
      maxMsgs: '500',
      maxMsgSize: 4096,
    })
    expect(res.stream).toBeTruthy()
    const got = await A.getStream(cid, N.stream)
    expect(got.stream?.config?.name).toBe(N.stream)
    expect(got.stream?.config?.subjects).toContain(N.wildcard)
    expect(got.stream?.config?.maxMsgSize).toBe(4096)
    expect(String(got.stream?.config?.storage ?? '0')).toBe('0')
  })

  test('CHK ListStreams includes our stream and filters $-internal streams', async () => {
    const list = await A.listStreams(cid)
    const names = (list.streams ?? []).map((s) => s.config?.name)
    expect(names).toContain(N.stream)
    expect(names.every((n) => !String(n).startsWith('$'))).toBe(true)
  })

  test('CHK PurgeStream removes messages, keeps stream', async () => {
    await A.publish(cid, N.subject('p1'), N.wildcard, 'a')
    await A.publish(cid, N.subject('p2'), N.wildcard, 'b')
    const before = await A.getStream(cid, N.stream)
    expect(Number(before.stream?.state?.msgs ?? '0')).toBeGreaterThanOrEqual(2)
    const purge = await A.purgeStream(cid, N.stream)
    expect(Number(purge.purged ?? '0')).toBeGreaterThanOrEqual(2)
    const after = await A.getStream(cid, N.stream)
    expect(Number(after.stream?.state?.msgs ?? '0')).toBe(0)
    expect(await A.streamExists(cid, N.stream)).toBe(true)
  })
})

test.describe('streams: seal is irreversible', () => {
  test('CHK sealed stream rejects data ops (purge & message delete)', async () => {
    await A.deleteStreamIfExists(cid, N.stream2)
    await A.createStream(cid, N.stream2, { subjects: [`${N.subjectRoot}2.>`], storage: 1 })
    const pub = await A.publish(cid, `${N.subjectRoot}2.x`, `${N.subjectRoot}2.>`, 'x')
    await A.sealStream(cid, N.stream2)
    const got = await A.getStream(cid, N.stream2)
    expect(got.stream?.config?.sealed).toBe(true)
    // The real irreversibility guarantee: no data may be removed from a sealed stream.
    await expect(A.purgeStream(cid, N.stream2)).rejects.toBeInstanceOf(A.ConnectError)
    await expect(A.deleteMessage(cid, N.stream2, Number(pub.sequence ?? '1'))).rejects.toBeInstanceOf(A.ConnectError)
    await A.deleteStreamIfExists(cid, N.stream2)
  })

  test('CHK [finding] config UPDATE on a sealed stream is permitted by NATS (not blocked)', async () => {
    // Documents a discrepancy discovery flagged as "all updates fail server-side":
    // in reality NATS accepts config changes on a sealed stream; only data ops are blocked.
    const s = `${N.stream}_SEALCFG`
    await A.deleteStreamIfExists(cid, s)
    await A.createStream(cid, s, { subjects: [`${N.subjectRoot}sc.>`], storage: 1, maxMsgs: '-1' })
    await A.sealStream(cid, s)
    await A.updateStream(cid, s, { maxMsgs: '10' })
    const got = await A.getStream(cid, s)
    expect(String(got.stream?.config?.maxMsgs)).toBe('10')
    await A.deleteStreamIfExists(cid, s)
  })
})

test.describe('streams: DeleteMessage bounds & deny_delete', () => {
  test('CHK [P2] DeleteMessage sequence must be > 0 (buf.validate uint64.gt=0)', async () => {
    await expect(A.deleteMessage(cid, N.stream, 0)).rejects.toBeInstanceOf(A.ConnectError)
  })

  test('CHK deny_delete stream refuses DeleteMessage', async () => {
    const s = `${N.stream}_DENY`
    await A.deleteStreamIfExists(cid, s)
    await A.createStream(cid, s, { subjects: [`${N.subjectRoot}deny.>`], denyDelete: true, storage: 1 })
    const pub = await A.publish(cid, `${N.subjectRoot}deny.x`, `${N.subjectRoot}deny.>`, 'x')
    expect(pub.error ?? '').toBe('')
    await expect(A.deleteMessage(cid, s, Number(pub.sequence ?? '1'))).rejects.toBeInstanceOf(A.ConnectError)
    await A.deleteStreamIfExists(cid, s)
  })
})

test.describe('streams: create validation', () => {
  test('CHK [P2] empty stream name rejected (buf.validate min_len=1)', async () => {
    await expect(A.createStream(cid, '', { subjects: ['x.>'] })).rejects.toBeInstanceOf(A.ConnectError)
  })
})

test.describe('consumers: lifecycle', () => {
  const stream = `${N.stream}_C`
  test.beforeAll(async () => {
    await A.deleteStreamIfExists(cid, stream)
    await A.createStream(cid, stream, { subjects: [`${N.subjectRoot}c.>`], storage: 1 })
  })
  test.afterAll(async () => {
    await A.deleteStreamIfExists(cid, stream)
  })

  test('CHK create durable pull consumer, list, delete', async () => {
    await A.createConsumer(cid, stream, 'durpull', { ackPolicy: 0, filterSubject: `${N.subjectRoot}c.>` })
    const list = await A.listConsumers(cid, stream)
    expect((list.consumers ?? []).map((c) => c.name)).toContain('durpull')
    await A.deleteConsumer(cid, stream, 'durpull')
    const after = await A.listConsumers(cid, stream)
    expect((after.consumers ?? []).map((c) => c.name)).not.toContain('durpull')
  })

  test('CHK pause with valid RFC3339, then resume', async () => {
    await A.createConsumer(cid, stream, 'pausable', { ackPolicy: 0 })
    const until = new Date(Date.now() + 60_000).toISOString()
    const p = await A.pauseConsumer(cid, stream, 'pausable', until)
    expect(p.paused).toBe(true)
    const r = await A.resumeConsumer(cid, stream, 'pausable')
    // proto3 false is omitted from Connect-JSON, so absent === not paused.
    expect(r.paused ?? false).toBe(false)
    await A.deleteConsumer(cid, stream, 'pausable')
  })

  test('CHK [P2] pause with malformed timestamp is rejected', async () => {
    await A.createConsumer(cid, stream, 'badpause', { ackPolicy: 0 })
    await expect(A.pauseConsumer(cid, stream, 'badpause', 'not-a-timestamp')).rejects.toBeInstanceOf(A.ConnectError)
    await A.deleteConsumer(cid, stream, 'badpause')
  })

  test('CHK [suspected-bug] CreateConsumer silently accepts malformed opt_start_time (deliverPolicy=byStartTime)', async () => {
    // Discovery slice 2 flagged: consumers.go drops a malformed opt_start_time
    // with no error, unlike PauseConsumer. Document actual behavior.
    let created: boolean
    try {
      await A.createConsumer(cid, stream, 'badstart', { deliverPolicy: 4, optStartTime: 'garbage', ackPolicy: 0 })
      created = true
    } catch {
      created = false
    }
    // This assertion encodes the EXPECTED-correct behavior: a malformed
    // start time should be rejected. If it is silently accepted, this fails
    // and is triaged as the suspected product bug.
    expect(created, 'malformed opt_start_time should be rejected like PauseConsumer').toBe(false)
    if (created) await A.deleteConsumer(cid, stream, 'badstart').catch(() => {})
  })
})
