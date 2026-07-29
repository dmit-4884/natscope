import { describe, it, expect } from 'vitest'
import { StreamConfig } from './StreamConfig'

describe('StreamConfig', () => {
  const defaultApiConfig = {
    retention: 'limits' as const,
    max_msgs: 1000,
    max_bytes: 1048576,
    max_age: 86400000000000, // 1 day in nanoseconds
  }

  describe('fromApi', () => {
    it('creates config with required fields', () => {
      const config = StreamConfig.fromApi(defaultApiConfig)
      expect(config.retention).toBe('limits')
      expect(config.maxMessages).toBe(1000)
      expect(config.maxBytes).toBe(1048576)
      expect(config.maxAge).toBe(86400000000000)
    })

    it('uses defaults for optional fields', () => {
      const config = StreamConfig.fromApi(defaultApiConfig)
      expect(config.storage).toBe('file')
      expect(config.numReplicas).toBe(1)
      expect(config.isSealed).toBe(false)
    })

    it('overrides optional fields when provided', () => {
      const config = StreamConfig.fromApi({
        ...defaultApiConfig,
        storage: 'memory',
        discard: 'new',
        num_replicas: 3,
        sealed: true,
        deny_delete: true,
        deny_purge: true,
      })
      expect(config.storage).toBe('memory')
      expect(config.numReplicas).toBe(3)
      expect(config.isSealed).toBe(true)
    })
  })

  describe('formattedMaxAge', () => {
    it('returns Unlimited for zero', () => {
      const config = StreamConfig.fromApi({ ...defaultApiConfig, max_age: 0 })
      expect(config.formattedMaxAge()).toBe('Unlimited')
    })

    it('returns Unlimited for negative', () => {
      const config = StreamConfig.fromApi({ ...defaultApiConfig, max_age: -1 })
      expect(config.formattedMaxAge()).toBe('Unlimited')
    })

    it('formats seconds', () => {
      const config = StreamConfig.fromApi({ ...defaultApiConfig, max_age: 30_000_000_000 })
      expect(config.formattedMaxAge()).toBe('30s')
    })

    it('formats minutes', () => {
      const config = StreamConfig.fromApi({ ...defaultApiConfig, max_age: 300_000_000_000 })
      expect(config.formattedMaxAge()).toBe('5m')
    })

    it('formats hours', () => {
      const config = StreamConfig.fromApi({ ...defaultApiConfig, max_age: 7200_000_000_000 })
      expect(config.formattedMaxAge()).toBe('2h')
    })

    it('formats days', () => {
      const config = StreamConfig.fromApi({ ...defaultApiConfig, max_age: 86400_000_000_000 })
      expect(config.formattedMaxAge()).toBe('1d')
    })
  })

  describe('formattedMaxBytes', () => {
    it('returns Unlimited for zero', () => {
      const config = StreamConfig.fromApi({ ...defaultApiConfig, max_bytes: 0 })
      expect(config.formattedMaxBytes()).toBe('Unlimited')
    })

    it('returns Unlimited for negative', () => {
      const config = StreamConfig.fromApi({ ...defaultApiConfig, max_bytes: -1 })
      expect(config.formattedMaxBytes()).toBe('Unlimited')
    })

    it('formats bytes', () => {
      const config = StreamConfig.fromApi({ ...defaultApiConfig, max_bytes: 500 })
      expect(config.formattedMaxBytes()).toBe('500 B')
    })

    it('formats kilobytes', () => {
      const config = StreamConfig.fromApi({ ...defaultApiConfig, max_bytes: 2048 })
      expect(config.formattedMaxBytes()).toBe('2.0 KB')
    })

    it('formats megabytes', () => {
      const config = StreamConfig.fromApi({ ...defaultApiConfig, max_bytes: 1048576 })
      expect(config.formattedMaxBytes()).toBe('1.0 MB')
    })

    it('formats gigabytes', () => {
      const config = StreamConfig.fromApi({ ...defaultApiConfig, max_bytes: 1073741824 })
      expect(config.formattedMaxBytes()).toBe('1.0 GB')
    })
  })

  describe('formattedMaxMessages', () => {
    it('returns Unlimited for zero', () => {
      const config = StreamConfig.fromApi({ ...defaultApiConfig, max_msgs: 0 })
      expect(config.formattedMaxMessages()).toBe('Unlimited')
    })

    it('returns Unlimited for negative', () => {
      const config = StreamConfig.fromApi({ ...defaultApiConfig, max_msgs: -1 })
      expect(config.formattedMaxMessages()).toBe('Unlimited')
    })

    it('formats small numbers', () => {
      const config = StreamConfig.fromApi({ ...defaultApiConfig, max_msgs: 500 })
      expect(config.formattedMaxMessages()).toBe('500')
    })

    it('formats thousands', () => {
      const config = StreamConfig.fromApi({ ...defaultApiConfig, max_msgs: 5000 })
      expect(config.formattedMaxMessages()).toBe('5.0K')
    })

    it('formats millions', () => {
      const config = StreamConfig.fromApi({ ...defaultApiConfig, max_msgs: 2000000 })
      expect(config.formattedMaxMessages()).toBe('2.0M')
    })
  })

  describe('retentionDisplayName', () => {
    it('returns Limits', () => {
      expect(StreamConfig.fromApi(defaultApiConfig).retentionDisplayName()).toBe('Limits')
    })

    it('returns Interest', () => {
      const config = StreamConfig.fromApi({ ...defaultApiConfig, retention: 'interest' })
      expect(config.retentionDisplayName()).toBe('Interest')
    })

    it('returns Work Queue', () => {
      const config = StreamConfig.fromApi({ ...defaultApiConfig, retention: 'workqueue' })
      expect(config.retentionDisplayName()).toBe('Work Queue')
    })
  })

  describe('hasLimits', () => {
    it('returns true when maxMessages is positive', () => {
      expect(StreamConfig.fromApi(defaultApiConfig).hasLimits()).toBe(true)
    })

    it('returns false when all limits are zero or negative', () => {
      const config = StreamConfig.fromApi({
        ...defaultApiConfig,
        max_msgs: -1,
        max_bytes: -1,
        max_age: 0,
      })
      expect(config.hasLimits()).toBe(false)
    })
  })
})
