import { test, expect } from '@playwright/test'
import * as A from './api-audit'
import { ns } from './fixtures-audit'

/**
 * Slice 10 — KV store (API-level): bucket CRUD, put/get (base64 value),
 * history depth, CAS revision, delete/purge markers, list keys.
 */
const N = ns('kv')
const BUCKET = N.kvBucket
let cid: string

test.beforeAll(async () => {
  cid = await A.getConnectionId('local')
  await A.deleteKVBucketIfExists(cid, BUCKET)
  await A.createKVBucket(cid, { bucket: BUCKET, history: 5, storage: 1 })
})
test.afterAll(async () => {
  await A.deleteKVBucketIfExists(cid, BUCKET)
})

test.describe('kv: bucket + key basics', () => {
  test('CHK bucket appears in list with its history depth', async () => {
    const buckets = (await A.listKVBuckets(cid)).buckets ?? []
    const b = buckets.find((x) => x.bucket === BUCKET)
    expect(b).toBeTruthy()
    expect(Number(b?.history ?? '0')).toBe(5)
  })

  test('CHK put then get round-trips the value (base64-decoded)', async () => {
    await A.putKVKey(cid, BUCKET, 'greeting', 'hello-kv')
    const got = await A.getKVKey(cid, BUCKET, 'greeting')
    expect(A.kvText(got.entry)).toBe('hello-kv')
    expect(got.entry?.operation).toBe('put')
  })

  test('CHK empty value is stored and retrievable', async () => {
    await A.putKVKey(cid, BUCKET, 'blank', '')
    const got = await A.getKVKey(cid, BUCKET, 'blank')
    expect(A.kvText(got.entry)).toBe('')
  })

  test('CHK list keys includes stored keys', async () => {
    const keys = (await A.listKVKeys(cid, BUCKET)).keys ?? []
    expect(keys).toContain('greeting')
  })
})

test.describe('kv: history & CAS', () => {
  test('CHK history returns bounded revisions oldest-first', async () => {
    const key = 'versioned'
    for (let i = 1; i <= 7; i++) await A.putKVKey(cid, BUCKET, key, `v${i}`)
    const hist = (await A.getKVKeyHistory(cid, BUCKET, key)).entries ?? []
    // Bucket history depth is 5, so at most 5 revisions are kept.
    expect(hist.length).toBeLessThanOrEqual(5)
    const revs = hist.map((h) => Number(h.revision))
    expect(revs).toEqual([...revs].sort((a, b) => a - b))
    expect(A.kvText(hist[hist.length - 1])).toBe('v7')
  })

  test('CHK KV CAS: a put with a wrong expected revision is rejected; correct revision succeeds', async () => {
    // Bug found by this audit and fixed: PutKVKey ignored the `revision` field
    // (always kv.Put). It now does a compare-and-swap (kv.Update) when revision
    // is non-zero, per the proto contract.
    const key = 'cas'
    const r1 = await A.putKVKey(cid, BUCKET, key, 'first')
    const rev = Number(r1.revision)
    await expect(A.putKVKey(cid, BUCKET, key, 'stale', rev + 99)).rejects.toBeInstanceOf(A.ConnectError)
    const r2 = await A.putKVKey(cid, BUCKET, key, 'second', rev)
    expect(Number(r2.revision)).toBe(rev + 1)
  })
})

test.describe('kv: delete & purge', () => {
  test('CHK delete adds a DEL marker; get then reports missing', async () => {
    const key = 'todelete'
    await A.putKVKey(cid, BUCKET, key, 'x')
    await A.deleteKVKey(cid, BUCKET, key)
    // After delete the key is not readable as a live value.
    await expect(A.getKVKey(cid, BUCKET, key)).rejects.toBeInstanceOf(A.ConnectError)
    // History still shows the del marker among operations.
    const ops = ((await A.getKVKeyHistory(cid, BUCKET, key)).entries ?? []).map((e) => e.operation)
    expect(ops.some((o) => /del|purge/i.test(String(o)))).toBe(true)
  })

  test('CHK purge collapses history for a key', async () => {
    const key = 'topurge'
    await A.putKVKey(cid, BUCKET, key, 'a')
    await A.putKVKey(cid, BUCKET, key, 'b')
    await A.purgeKVKey(cid, BUCKET, key)
    const hist = (await A.getKVKeyHistory(cid, BUCKET, key)).entries ?? []
    // After purge only the purge marker remains.
    expect(hist.length).toBeLessThanOrEqual(1)
  })
})

test.describe('kv: validation', () => {
  test('CHK [P2] operations on a missing bucket error', async () => {
    await expect(A.getKVKey(cid, 'audit-kv-nope-xyz', 'k')).rejects.toBeInstanceOf(A.ConnectError)
  })
})
