import { describe, it, expect } from 'vitest'
import { EMPTY_PURGE_FORM, type PurgeFormState, buildPurgeRequest, getPurgeFormError } from './purgeOptions'

function form(over: Partial<PurgeFormState> = {}): PurgeFormState {
  return { ...EMPTY_PURGE_FORM, ...over }
}

describe('getPurgeFormError', () => {
  it('accepts a plain full purge', () => {
    expect(getPurgeFormError(form())).toBeUndefined()
    expect(getPurgeFormError(form({ filter: 'orders.>' }))).toBeUndefined()
  })

  it('requires a positive integer for the sequence mode', () => {
    expect(getPurgeFormError(form({ mode: 'sequence' }))).toBeDefined()
    expect(getPurgeFormError(form({ mode: 'sequence', sequence: '0' }))).toBeDefined()
    expect(getPurgeFormError(form({ mode: 'sequence', sequence: '-4' }))).toBeDefined()
    expect(getPurgeFormError(form({ mode: 'sequence', sequence: '1.5' }))).toBeDefined()
    expect(getPurgeFormError(form({ mode: 'sequence', sequence: 'abc' }))).toBeDefined()
    expect(getPurgeFormError(form({ mode: 'sequence', sequence: ' 12 ' }))).toBeUndefined()
  })

  it('requires a positive integer for the keep mode', () => {
    expect(getPurgeFormError(form({ mode: 'keep' }))).toBeDefined()
    expect(getPurgeFormError(form({ mode: 'keep', keep: '0' }))).toBeDefined()
    expect(getPurgeFormError(form({ mode: 'keep', keep: '5' }))).toBeUndefined()
  })

  it('ignores the value of the mode that is not selected', () => {
    expect(getPurgeFormError(form({ mode: 'keep', keep: '5', sequence: 'garbage' }))).toBeUndefined()
  })
})

describe('buildPurgeRequest', () => {
  it('maps a full purge to no options', () => {
    expect(buildPurgeRequest(form())).toBeUndefined()
    expect(buildPurgeRequest(form({ filter: '   ' }))).toBeUndefined()
  })

  it('maps a subject-only purge', () => {
    expect(buildPurgeRequest(form({ filter: ' orders.eu ' }))).toEqual({ filter: 'orders.eu' })
  })

  it('maps a sequence purge and never sends keep alongside it', () => {
    expect(buildPurgeRequest(form({ mode: 'sequence', sequence: '1000', keep: '5' }))).toEqual({ seq: 1000 })
  })

  it('maps a keep purge and never sends seq alongside it', () => {
    expect(buildPurgeRequest(form({ mode: 'keep', keep: '5', sequence: '1000' }))).toEqual({ keep: 5 })
  })

  it('combines a filter with either limit', () => {
    expect(buildPurgeRequest(form({ filter: 'orders.eu', mode: 'sequence', sequence: '42' }))).toEqual({
      filter: 'orders.eu',
      seq: 42,
    })
    expect(buildPurgeRequest(form({ filter: 'orders.eu', mode: 'keep', keep: '3' }))).toEqual({
      filter: 'orders.eu',
      keep: 3,
    })
  })

  it('returns nothing while the form is invalid', () => {
    expect(buildPurgeRequest(form({ filter: 'orders.eu', mode: 'keep', keep: '0' }))).toBeUndefined()
  })
})
