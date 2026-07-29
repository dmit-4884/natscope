import { test, expect } from '@playwright/test'
import * as A from './api-audit'

/**
 * Slice 12 — Settings (API-level). Runs against the isolated audit data dir,
 * and resets to defaults in afterAll so no surprising state is left behind.
 */
interface Settings {
  settings?: {
    messages?: Record<string, unknown>
    behavior?: Record<string, unknown>
    display?: Record<string, unknown>
    live?: Record<string, unknown>
  }
}

test.afterAll(async () => {
  await A.resetSettings().catch(() => {})
})

test.describe('settings: get/update/reset', () => {
  test('CHK GetSettings returns a settings object with defaults', async () => {
    const s = (await A.getSettings()) as Settings
    expect(s.settings).toBeTruthy()
  })

  test('CHK update behavior toggle round-trips', async () => {
    await A.updateSettings({ behavior: { confirmDeleteConsumer: false } })
    const s = (await A.getSettings()) as Settings
    expect(s.settings?.behavior?.confirmDeleteConsumer).toBe(false)
  })

  test('CHK update message settings round-trips', async () => {
    await A.updateSettings({ messages: { defaultPageSize: 200, defaultDirection: 'forward' } })
    const s = (await A.getSettings()) as Settings
    expect(Number(s.settings?.messages?.defaultPageSize ?? '0')).toBe(200)
  })

  test('CHK ResetSettings restores defaults', async () => {
    await A.updateSettings({ messages: { defaultPageSize: 500 } })
    await A.resetSettings()
    const s = (await A.getSettings()) as Settings
    // Default page size is not 500 after reset (proto default omitted or reverts).
    const pageSize = Number(s.settings?.messages?.defaultPageSize ?? '0')
    expect(pageSize).not.toBe(500)
  })
})

test.describe('settings: validation bounds (P2)', () => {
  test('CHK defaultPageSize below 1 rejected (int32 gte=1)', async () => {
    await expect(A.updateSettings({ messages: { defaultPageSize: 0 } })).rejects.toBeInstanceOf(A.ConnectError)
  })
  test('CHK defaultPageSize above 10000 rejected (int32 lte=10000)', async () => {
    await expect(A.updateSettings({ messages: { defaultPageSize: 10001 } })).rejects.toBeInstanceOf(A.ConnectError)
  })
  test('CHK defaultExportFormat not in {json,ndjson,csv} rejected (string.in)', async () => {
    await expect(A.updateSettings({ messages: { defaultExportFormat: 'xml' } })).rejects.toBeInstanceOf(A.ConnectError)
  })
  test('CHK maxDisplayRate above 10000 rejected (live int32 lte=10000)', async () => {
    await expect(A.updateSettings({ live: { maxDisplayRate: 10001 } })).rejects.toBeInstanceOf(A.ConnectError)
  })
  test('CHK jsonIndentSize above 8 rejected (display int32 lte=8)', async () => {
    await expect(A.updateSettings({ display: { jsonIndentSize: 9 } })).rejects.toBeInstanceOf(A.ConnectError)
  })
  test('CHK publishTimeoutSec above 3600 rejected (publish int32 lte=3600)', async () => {
    await expect(A.updateSettings({ publish: { publishTimeoutSec: 3601 } })).rejects.toBeInstanceOf(A.ConnectError)
  })
  test('CHK boundary defaultPageSize=1 and =10000 accepted', async () => {
    await A.updateSettings({ messages: { defaultPageSize: 1 } })
    await A.updateSettings({ messages: { defaultPageSize: 10000 } })
    const s = (await A.getSettings()) as Settings
    expect(Number(s.settings?.messages?.defaultPageSize ?? '0')).toBe(10000)
  })
})
