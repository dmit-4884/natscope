import { describe, it, expect } from 'vitest'
import { isValidHeaderName } from './headerValidation'

describe('isValidHeaderName', () => {
  it('accepts common header names', () => {
    expect(isValidHeaderName('X-Ok')).toBe(true)
    expect(isValidHeaderName('Nats-Msg-Id')).toBe(true)
    expect(isValidHeaderName('X-Trace-1')).toBe(true)
  })

  it('rejects an empty name', () => {
    expect(isValidHeaderName('')).toBe(false)
  })

  it('rejects a name with a space', () => {
    expect(isValidHeaderName('a b')).toBe(false)
  })

  it('rejects a name with a colon', () => {
    expect(isValidHeaderName('a:b')).toBe(false)
  })

  it('rejects a non-ASCII name', () => {
    expect(isValidHeaderName('Ключ')).toBe(false)
  })

  it('rejects a name with CR/LF', () => {
    expect(isValidHeaderName('a\r\nb')).toBe(false)
  })
})
