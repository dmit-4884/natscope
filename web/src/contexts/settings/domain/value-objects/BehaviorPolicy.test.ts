import { describe, it, expect } from 'vitest'
import { BehaviorPolicy, BEHAVIOR_POLICY_DEFAULTS } from './BehaviorPolicy'

describe('BehaviorPolicy', () => {
  it('default() has every confirmation ON and secure-delete OFF', () => {
    const p = BehaviorPolicy.default()
    expect(p.toObject()).toEqual(BEHAVIOR_POLICY_DEFAULTS)
    expect(p.confirmDeleteConsumer).toBe(true)
    expect(p.confirmDeleteMessage).toBe(true)
    expect(p.secureDeleteDefault).toBe(false)
  })

  it('fromPartial collapses nil confirm fields to ON (the safe default)', () => {
    const p = BehaviorPolicy.fromPartial({ confirmDeleteMessage: false })
    expect(p.confirmDeleteMessage).toBe(false)
    // Unspecified toggles remain ON.
    expect(p.confirmDeleteConsumer).toBe(true)
    expect(p.confirmDeleteKvKey).toBe(true)
  })

  it('fromPartial(null) returns defaults', () => {
    expect(BehaviorPolicy.fromPartial(null).toObject()).toEqual(BEHAVIOR_POLICY_DEFAULTS)
  })

  it('shouldConfirm maps action keys to their toggle', () => {
    const p = BehaviorPolicy.fromPartial({ confirmDeleteObject: false })
    expect(p.shouldConfirm('deleteObject')).toBe(false)
    expect(p.shouldConfirm('deleteConsumer')).toBe(true)
    expect(p.shouldConfirm('purgeKvHistory')).toBe(true)
  })

  it('merge overrides only provided fields', () => {
    const base = BehaviorPolicy.fromPartial({ confirmDeleteConsumer: false })
    const r = base.merge({ confirmDeleteMessage: false, secureDeleteDefault: true })
    expect(r.isOk()).toBe(true)
    if (r.isOk()) {
      expect(r.value.confirmDeleteConsumer).toBe(false) // preserved
      expect(r.value.confirmDeleteMessage).toBe(false) // overridden
      expect(r.value.secureDeleteDefault).toBe(true)
    }
  })
})
