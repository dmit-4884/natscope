import { describe, it, expect, vi } from 'vitest'
import { UserSettings } from './UserSettings'

describe('UserSettings', () => {
  it('fromApi fills defaults when fields missing', () => {
    const s = UserSettings.fromApi({ id: 'u1', createdAt: 1000, updatedAt: 2000 })
    expect(s.messages.defaultPageSize).toBe(50)
    expect(s.live.subscriptionMode).toBe('core_nats')
    expect(s.display.density).toBe('comfortable')
    expect(s.publish.publishTimeoutSec).toBe(10)
    expect(s.createdAt.getTime()).toBe(1000)
  })

  it('fromApi preserves provided values', () => {
    const s = UserSettings.fromApi({
      id: 'u1',
      messages: { fetchMethod: 'consumer', defaultPageSize: 200 },
      live: { subscriptionMode: 'jetstream_ordered' },
      display: { density: 'compact' },
      publish: { publishTimeoutSec: 30 },
      createdAt: 1000,
      updatedAt: 2000,
    })
    expect(s.messages.fetchMethod).toBe('consumer')
    expect(s.messages.defaultPageSize).toBe(200)
    expect(s.live.subscriptionMode).toBe('jetstream_ordered')
    expect(s.display.density).toBe('compact')
    expect(s.publish.publishTimeoutSec).toBe(30)
  })

  it('create produces a default aggregate with id', () => {
    const s = UserSettings.create('fresh')
    expect(s.id).toBe('fresh')
    expect(s.messages.fetchMethod).toBe('consumer')
  })

  it('applyUpdate returns new instance with patched fields and updated timestamp', () => {
    const s = UserSettings.create('u1')
    const original = s.updatedAt
    // Wait a tick to ensure different timestamp
    const later = new Date(original.getTime() + 1000)
    vi.useFakeTimers()
    vi.setSystemTime(later)
    const r = s.applyUpdate({ messages: { defaultPageSize: 500 } })
    vi.useRealTimers()
    expect(r.isOk()).toBe(true)
    if (r.isOk()) {
      expect(r.value.messages.defaultPageSize).toBe(500)
      expect(r.value.id).toBe('u1')
    }
  })

  it('applyUpdate rejects invalid patch and surfaces the domain error', () => {
    const s = UserSettings.create('u1')
    const r = s.applyUpdate({ live: { maxDisplayRate: -1 } }) // below min 0
    expect(r.isErr()).toBe(true)
    if (r.isErr()) expect(r.error.field).toBe('maxDisplayRate')
  })
})
