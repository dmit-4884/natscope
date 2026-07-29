import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { helpers, processHelpers } from './helpers'

describe('helpers', () => {
  describe('helpers array', () => {
    it('has UUID helper', () => {
      const uuid = helpers.find((h) => h.name === 'UUID')
      expect(uuid).toBeDefined()
      expect(uuid!.template).toBe('{{uuid}}')
      // UUID format
      const value = uuid!.generate()
      expect(value).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i)
    })

    it('has Timestamp helper', () => {
      const ts = helpers.find((h) => h.name === 'Timestamp')
      expect(ts).toBeDefined()
      const value = ts!.generate()
      expect(new Date(value).toString()).not.toBe('Invalid Date')
    })

    it('has Unix helper', () => {
      const unix = helpers.find((h) => h.name === 'Unix')
      expect(unix).toBeDefined()
      const value = Number(unix!.generate())
      expect(value).toBeGreaterThan(0)
    })

    it('has Random Int helper', () => {
      const randInt = helpers.find((h) => h.name === 'Random Int')
      expect(randInt).toBeDefined()
      const value = Number(randInt!.generate())
      expect(value).toBeGreaterThanOrEqual(1)
      expect(value).toBeLessThanOrEqual(1000)
    })

    it('has Random String helper', () => {
      const randStr = helpers.find((h) => h.name === 'Random String')
      expect(randStr).toBeDefined()
      const value = randStr!.generate()
      expect(value.length).toBe(8)
    })

    it('has Date helper', () => {
      const date = helpers.find((h) => h.name === 'Date')
      expect(date).toBeDefined()
      expect(date!.generate()).toMatch(/^\d{4}-\d{2}-\d{2}$/)
    })

    it('has Time helper', () => {
      const time = helpers.find((h) => h.name === 'Time')
      expect(time).toBeDefined()
      expect(time!.generate()).toMatch(/^\d{2}:\d{2}:\d{2}$/)
    })
  })

  describe('processHelpers', () => {
    // Mock crypto.randomUUID and Date.now for deterministic tests
    let uuidCounter: number

    beforeEach(() => {
      uuidCounter = 0
      vi.spyOn(crypto, 'randomUUID').mockImplementation(
        () => `00000000-0000-0000-0000-${String(++uuidCounter).padStart(12, '0')}` as `${string}-${string}-${string}-${string}-${string}`
      )
    })

    afterEach(() => {
      vi.restoreAllMocks()
    })

    it('replaces UUID template', () => {
      const result = processHelpers('id: {{uuid}}')
      expect(result).toContain('00000000-0000-0000-0000-000000000001')
    })

    it('replaces multiple templates with different values', () => {
      const result = processHelpers('{{uuid}} {{uuid}}')
      // Each replacement should generate a new value
      expect(result).toContain('00000000-0000-0000-0000-000000000001')
    })

    it('shared helpers ($ prefix) reuse the same value', () => {
      const result = processHelpers('{{$uuid}} and {{$uuid}}')
      // Both should have the same value
      const parts = result.split(' and ')
      expect(parts[0]).toBe(parts[1])
    })

    it('passes shared values between calls', () => {
      const shared: Record<string, string> = {}
      const result1 = processHelpers('id: {{$uuid}}', shared)
      const result2 = processHelpers('ref: {{$uuid}}', shared)
      // Both should contain the same UUID
      const uuid1 = result1.replace('id: ', '')
      const uuid2 = result2.replace('ref: ', '')
      expect(uuid1).toBe(uuid2)
    })

    it('returns text unchanged when no templates', () => {
      expect(processHelpers('no templates here')).toBe('no templates here')
    })

    it('handles JSON context with quotes', () => {
      const result = processHelpers('{"id": {{uuid}}}')
      // In JSON context (after :), value should be quoted
      expect(result).toMatch(/"id":\s*"[^"]+"/i)
    })
  })
})
