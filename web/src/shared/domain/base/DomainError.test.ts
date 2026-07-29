import { describe, it, expect } from 'vitest'
import { DomainError } from './DomainError'

describe('DomainError', () => {
  it('constructs with required init', () => {
    const err = new DomainError({ kind: 'validation', message: 'bad' })
    expect(err.kind).toBe('validation')
    expect(err.message).toBe('bad')
    expect(err.name).toBe('DomainError')
    expect(err).toBeInstanceOf(DomainError)
    expect(err).toBeInstanceOf(Error)
  })

  it('validation factory sets kind and field', () => {
    const err = DomainError.validation('too short', 'name')
    expect(err.kind).toBe('validation')
    expect(err.field).toBe('name')
  })

  it('invariant, notFound, conflict, unauthorized factories set correct kinds', () => {
    expect(DomainError.invariant('x').kind).toBe('invariant')
    expect(DomainError.notFound('User').message).toBe('User not found')
    expect(DomainError.conflict('dup').kind).toBe('conflict')
    expect(DomainError.unauthorized().kind).toBe('unauthorized')
  })

  it('wrap returns existing DomainError unchanged', () => {
    const orig = DomainError.validation('keep')
    const wrapped = DomainError.wrap(orig)
    expect(wrapped).toBe(orig)
  })

  it('wrap turns Error into DomainError', () => {
    const src = new Error('boom')
    const wrapped = DomainError.wrap(src, 'invariant')
    expect(wrapped.kind).toBe('invariant')
    expect(wrapped.message).toBe('boom')
    expect(wrapped.reason).toBe(src)
  })

  it('wrap turns non-Error into DomainError', () => {
    const wrapped = DomainError.wrap('string-error')
    expect(wrapped.kind).toBe('unknown')
    expect(wrapped.message).toBe('string-error')
  })

  it('toJSON returns serializable form', () => {
    const err = DomainError.validation('nope', 'email')
    expect(err.toJSON()).toEqual({ kind: 'validation', message: 'nope', field: 'email' })
  })
})
