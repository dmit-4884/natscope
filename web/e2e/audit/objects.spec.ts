import { test, expect } from '@playwright/test'
import * as A from './api-audit'
import { ns } from './fixtures-audit'

/**
 * Slice 11 — Object store (API-level): bucket CRUD, put/get roundtrip with
 * digest, multi-chunk large object, list, delete, seal-then-put rejection.
 */
const N = ns('objects')
const BUCKET = N.objBucket
let cid: string

test.beforeAll(async () => {
  cid = await A.getConnectionId('local')
  await A.deleteObjectBucketIfExists(cid, BUCKET)
  await A.createObjectBucket(cid, { bucket: BUCKET, storage: 1 })
})
test.afterAll(async () => {
  await A.deleteObjectBucketIfExists(cid, BUCKET)
})

test.describe('objects: put/get', () => {
  test('CHK put then get round-trips bytes and reports size + digest', async () => {
    const body = 'hello-object-store-payload'
    const put = await A.putObject(cid, BUCKET, 'greeting.txt', body, { description: 'greeting' })
    expect(Number(put.info?.size)).toBe(Buffer.byteLength(body))
    expect(String(put.info?.digest ?? '')).toContain('SHA-256=')
    const got = await A.getObject(cid, BUCKET, 'greeting.txt')
    expect(A.unb64(got.data ?? '').toString('utf8')).toBe(body)
  })

  test('CHK multi-chunk large object (>128KB) has chunks > 1 and round-trips', async () => {
    const big = Buffer.alloc(300 * 1024, 7) // 300 KiB
    const put = await A.putObject(cid, BUCKET, 'big.bin', big)
    expect(Number(put.info?.size)).toBe(big.length)
    expect(Number(put.info?.chunks ?? '0')).toBeGreaterThan(1)
    const got = await A.getObject(cid, BUCKET, 'big.bin')
    expect(A.unb64(got.data ?? '').length).toBe(big.length)
  })
})

test.describe('objects: list & delete', () => {
  test('CHK list shows stored objects; delete removes it', async () => {
    await A.putObject(cid, BUCKET, 'temp.txt', 'x')
    let names = ((await A.listObjects(cid, BUCKET)).objects ?? []).filter((o) => !o.deleted).map((o) => o.name)
    expect(names).toContain('temp.txt')
    await A.deleteObject(cid, BUCKET, 'temp.txt')
    names = ((await A.listObjects(cid, BUCKET)).objects ?? []).filter((o) => !o.deleted).map((o) => o.name)
    expect(names).not.toContain('temp.txt')
  })
})

test.describe('objects: seal', () => {
  test('CHK sealing a bucket rejects further puts', async () => {
    const sealed = `${BUCKET}-sealed`
    await A.deleteObjectBucketIfExists(cid, sealed)
    await A.createObjectBucket(cid, { bucket: sealed, storage: 1 })
    await A.putObject(cid, sealed, 'before.txt', 'ok')
    await A.sealObjectBucket(cid, sealed)
    await expect(A.putObject(cid, sealed, 'after.txt', 'nope')).rejects.toBeInstanceOf(A.ConnectError)
    await A.deleteObjectBucketIfExists(cid, sealed)
  })
})
