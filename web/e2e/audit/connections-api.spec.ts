import { test, expect } from '@playwright/test'
import * as A from './api-audit'

/**
 * Slice 1 — Connections & auth (API-level). Uses only throwaway connections
 * with the AUDIT_CONN_ prefix; never touches the `local` connection.
 */
const PREFIX = 'AUDIT_CONN_'
const name = (s: string) => `${PREFIX}${s}`

async function cleanup() {
  for (const c of await A.listConnections()) {
    if (c.name.startsWith(PREFIX)) await A.deleteConnection(c.id).catch(() => {})
  }
}
test.beforeAll(cleanup)
test.afterAll(cleanup)

test.describe('connections: CRUD lifecycle', () => {
  test('CHK create, list, update urls, duplicate, delete', async () => {
    const created = await A.createConnection({ name: name('crud'), urls: ['nats://localhost:4222'] })
    const id = created.connection!.id
    expect((await A.listConnections()).some((c) => c.id === id)).toBe(true)

    await A.updateConnection({ id, urls: ['nats://localhost:4222', 'nats://127.0.0.1:4222'] })
    const listed = (await A.listConnections()).find((c) => c.id === id)
    expect(listed?.urls?.length).toBe(2)

    const dup = await A.duplicateConnection(id, name('crud-copy'))
    expect(dup.connection?.id).not.toBe(id)

    await A.deleteConnection(id)
    expect((await A.listConnections()).some((c) => c.id === id)).toBe(false)
  })
})

test.describe('connections: secret redaction (security P0)', () => {
  test('CHK stored password is never echoed; only has_password presence flag', async () => {
    const created = await A.createConnection({
      name: name('secret'),
      urls: ['nats://localhost:4222'],
      auth: { method: 'AUTH_METHOD_USER_PASSWORD', username: 'u', password: 'super-secret-pw' },
    })
    const conn = created.connection as Record<string, unknown> & { auth?: Record<string, unknown> }
    expect(conn.auth?.password, 'raw password must not be returned').toBeUndefined()
    expect(conn.auth?.hasPassword).toBe(true)
    // Re-fetch via list: still redacted.
    const relisted = (await A.listConnections()).find((c) => c.id === created.connection!.id) as { auth?: Record<string, unknown> }
    expect(relisted.auth?.password).toBeUndefined()
  })
})

test.describe('connections: TestConnection', () => {
  test('CHK ad-hoc test against the running server succeeds with JetStream', async () => {
    const r = await A.testConnection({ urls: ['nats://localhost:4222'] })
    expect(r.success).toBe(true)
    expect(r.jetstreamEnabled).toBe(true)
  })

  test('CHK test against a dead port fails with an error', async () => {
    const r = await A.testConnection({ urls: ['nats://localhost:4999'], connectTimeout: '2s' })
    expect(r.success ?? false).toBe(false)
    expect(String(r.error ?? '')).not.toBe('')
  })
})

test.describe('connections: validation bounds', () => {
  test('CHK [P2] empty name rejected (min_len=1)', async () => {
    await expect(A.createConnection({ name: '', urls: ['nats://localhost:4222'] })).rejects.toBeInstanceOf(A.ConnectError)
  })
  test('CHK [P2] no urls rejected (repeated.min_items=1)', async () => {
    await expect(A.createConnection({ name: name('nourl'), urls: [] })).rejects.toBeInstanceOf(A.ConnectError)
  })
  test('CHK [P2] update with non-UUID id rejected (string.uuid)', async () => {
    await expect(A.updateConnection({ id: 'not-a-uuid', name: name('x') })).rejects.toBeInstanceOf(A.ConnectError)
  })
})

test.describe('connections: regression — whitespace-name validation (fixed)', () => {
  test('CHK whitespace-only name is rejected (was: passed min_len=1 then trimmed to empty)', async () => {
    // Bug found by this audit and fixed: buf.validate min_len=1 runs on the raw
    // request, so "   " passed and the normalizer then trimmed it to an empty
    // stored name. The service now re-checks the trimmed name → invalid_argument.
    await expect(A.createConnection({ name: '   ', urls: ['nats://localhost:4222'] })).rejects.toBeInstanceOf(A.ConnectError)
  })
})
