import { describe, it, expect, vi } from 'vitest'
import { Result, combineResults } from './Result'

describe('Result', () => {
  describe('ok / err construction', () => {
    it('creates an Ok with a value', () => {
      const r = Result.ok<number>(42)
      expect(r.isOk()).toBe(true)
      expect(r.isErr()).toBe(false)
      expect(r.value).toBe(42)
    })

    it('creates an Err with an error', () => {
      const err = new Error('boom')
      const r = Result.err<number, Error>(err)
      expect(r.isOk()).toBe(false)
      expect(r.isErr()).toBe(true)
      expect(r.error).toBe(err)
    })
  })

  describe('value / error access guards', () => {
    it('throws when accessing value of Err', () => {
      const r = Result.err<number, Error>(new Error('nope'))
      expect(() => r.value).toThrow('Cannot get value from an error Result')
    })

    it('throws when accessing error of Ok', () => {
      const r = Result.ok<number>(1)
      expect(() => r.error).toThrow('Cannot get error from a successful Result')
    })
  })

  describe('unwrapOr / unwrapOrElse', () => {
    it('unwrapOr returns value when Ok', () => {
      expect(Result.ok(5).unwrapOr(0)).toBe(5)
    })

    it('unwrapOr returns default when Err', () => {
      expect(Result.err<number, Error>(new Error()).unwrapOr(42)).toBe(42)
    })

    it('unwrapOrElse returns mapped error when Err', () => {
      const r = Result.err<number, Error>(new Error('x'))
      expect(r.unwrapOrElse((e) => e.message.length)).toBe(1)
    })
  })

  describe('map / mapErr / flatMap', () => {
    it('maps value when Ok', () => {
      const r = Result.ok<number>(3).map((n) => n * 2)
      expect(r.value).toBe(6)
    })

    it('skips map when Err', () => {
      const fn = vi.fn((n: number) => n * 2)
      const r = Result.err<number, Error>(new Error('no')).map(fn)
      expect(fn).not.toHaveBeenCalled()
      expect(r.isErr()).toBe(true)
    })

    it('mapErr maps the error when Err', () => {
      const r = Result.err<number, Error>(new Error('a')).mapErr((e) => e.message + '!')
      expect(r.error).toBe('a!')
    })

    it('flatMap chains when Ok', () => {
      const r = Result.ok<number>(2).flatMap((n) => Result.ok<number>(n + 10))
      expect(r.value).toBe(12)
    })

    it('flatMap propagates Err', () => {
      const r = Result.err<number, Error>(new Error('bad')).flatMap((n) => Result.ok<number>(n))
      expect(r.isErr()).toBe(true)
    })
  })

  describe('tap / tapErr', () => {
    it('tap fires on Ok', () => {
      const spy = vi.fn()
      Result.ok<number>(9).tap(spy)
      expect(spy).toHaveBeenCalledWith(9)
    })

    it('tap skipped on Err', () => {
      const spy = vi.fn()
      Result.err<number, Error>(new Error()).tap(spy)
      expect(spy).not.toHaveBeenCalled()
    })

    it('tapErr fires on Err', () => {
      const spy = vi.fn()
      Result.err<number, Error>(new Error('x')).tapErr(spy)
      expect(spy).toHaveBeenCalled()
    })
  })

  describe('fold', () => {
    it('runs onOk on Ok', () => {
      const x = Result.ok<number>(3).fold((v) => v + 1, () => -1)
      expect(x).toBe(4)
    })

    it('runs onErr on Err', () => {
      const x = Result.err<number, Error>(new Error()).fold((v) => v, () => -1)
      expect(x).toBe(-1)
    })
  })

  describe('combineResults', () => {
    it('combines all Ok values', () => {
      const r = combineResults([Result.ok(1), Result.ok(2), Result.ok(3)])
      expect(r.isOk()).toBe(true)
      expect(r.value).toEqual([1, 2, 3])
    })

    it('returns first Err', () => {
      const err = new Error('second')
      const r = combineResults([Result.ok<number, Error>(1), Result.err<number, Error>(err), Result.ok<number, Error>(3)])
      expect(r.isErr()).toBe(true)
      expect(r.error).toBe(err)
    })

    it('returns empty Ok for empty input', () => {
      const r = combineResults<number, Error>([])
      expect(r.isOk()).toBe(true)
      expect(r.value).toEqual([])
    })
  })
})
