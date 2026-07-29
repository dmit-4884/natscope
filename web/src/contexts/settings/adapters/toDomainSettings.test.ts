import { describe, it, expect } from 'vitest'
import { create } from '@bufbuild/protobuf'
import { UserSettingsSchema } from '@/gen/types/settings/user_settings_pb'
import { toDomainSettings } from './toDomainSettings'

describe('toDomainSettings', () => {
  it('converts a fully-populated proto to a domain aggregate', () => {
    const proto = create(UserSettingsSchema, {
      id: 'u1',
      messages: {
        fetchMethod: 'consumer',
        defaultPageSize: 200,
        defaultDirection: 'forward',
      },
      live: {
        subscriptionMode: 'jetstream_ordered',
        maxDisplayRate: 1000,
      },
      display: {
        density: 'compact',
        defaultViewMode: 'realtime',
        payloadPreviewLen: 100,
        timestampFormat: 'iso',
        jsonIndentSize: 4,
        autoScrollLive: false,
      },
      publish: { publishTimeoutSec: 30 },
    })
    const s = toDomainSettings(proto)
    expect(s.id).toBe('u1')
    expect(s.messages.fetchMethod).toBe('consumer')
    expect(s.messages.defaultPageSize).toBe(200)
    expect(s.live.subscriptionMode).toBe('jetstream_ordered')
    expect(s.live.maxDisplayRate).toBe(1000)
    expect(s.display.jsonIndentSize).toBe(4)
    expect(s.display.autoScrollLive).toBe(false)
    expect(s.publish.publishTimeoutSec).toBe(30)
  })

  it('applies defaults for missing sub-messages', () => {
    const proto = create(UserSettingsSchema, { id: 'u1' })
    const s = toDomainSettings(proto)
    expect(s.messages.fetchMethod).toBe('consumer')
    expect(s.live.subscriptionMode).toBe('core_nats')
    expect(s.display.density).toBe('comfortable')
    expect(s.publish.publishTimeoutSec).toBe(10)
  })

  it('handles partial sub-messages (only some fields present)', () => {
    const proto = create(UserSettingsSchema, {
      id: 'u1',
      messages: { fetchMethod: 'consumer' },
    })
    const s = toDomainSettings(proto)
    expect(s.messages.fetchMethod).toBe('consumer')
    expect(s.messages.defaultPageSize).toBe(50) // default
  })
})
