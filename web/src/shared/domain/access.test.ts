import { describe, it, expect } from 'vitest'
import { describePermission, isDenied } from './access'

describe('describePermission', () => {
  it('names the operation and the subject', () => {
    expect(describePermission({ operation: 'publish', subject: '$SRV.INFO' })).toBe('publish to $SRV.INFO')
    expect(describePermission({ operation: 'subscribe', subject: '_INBOX.>' })).toBe('subscribe to _INBOX.>')
  })
})

describe('isDenied', () => {
  it('is true only for a denied check', () => {
    expect(isDenied({ status: 'denied', operation: 'publish', subject: 'x' })).toBe(true)
    expect(isDenied({ status: 'allowed', operation: 'publish', subject: 'x' })).toBe(false)
    expect(isDenied(undefined)).toBe(false)
  })
})
