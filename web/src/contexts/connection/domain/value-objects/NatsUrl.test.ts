import { describe, it, expect } from 'vitest'
import { NatsUrl } from './NatsUrl'

describe('NatsUrl', () => {
  describe('create', () => {
    it('parses valid nats URL', () => {
      const result = NatsUrl.create('nats://localhost:4222')
      expect(result.isOk()).toBe(true)
      expect(result.value.protocol).toBe('nats')
      expect(result.value.host).toBe('localhost')
      expect(result.value.port).toBe(4222)
      expect(result.value.value).toBe('nats://localhost:4222')
    })

    it('parses a URL with embedded credentials', () => {
      const result = NatsUrl.create('nats://user:pass@localhost:4222')
      expect(result.isOk()).toBe(true)
      expect(result.value.host).toBe('localhost')
      expect(result.value.port).toBe(4222)
    })

    it('parses tls URL', () => {
      const result = NatsUrl.create('tls://secure.nats.io:4443')
      expect(result.isOk()).toBe(true)
      expect(result.value.protocol).toBe('tls')
      expect(result.value.host).toBe('secure.nats.io')
      expect(result.value.port).toBe(4443)
    })

    it('parses ws URL', () => {
      const result = NatsUrl.create('ws://ws.nats.io:8080')
      expect(result.isOk()).toBe(true)
      expect(result.value.protocol).toBe('ws')
    })

    it('parses wss URL', () => {
      const result = NatsUrl.create('wss://secure-ws.nats.io:443')
      expect(result.isOk()).toBe(true)
      expect(result.value.protocol).toBe('wss')
    })

    it('defaults port to 4222 when omitted', () => {
      const result = NatsUrl.create('nats://localhost')
      expect(result.isOk()).toBe(true)
      expect(result.value.port).toBe(4222)
    })

    it('trims whitespace', () => {
      const result = NatsUrl.create('  nats://localhost:4222  ')
      expect(result.isOk()).toBe(true)
      expect(result.value.host).toBe('localhost')
    })

    it('rejects empty string', () => {
      const result = NatsUrl.create('')
      expect(result.isErr()).toBe(true)
      expect(result.error.code).toBe('EMPTY')
    })

    it('rejects whitespace-only string', () => {
      const result = NatsUrl.create('   ')
      expect(result.isErr()).toBe(true)
      expect(result.error.code).toBe('EMPTY')
    })

    it('rejects invalid format', () => {
      const result = NatsUrl.create('http://localhost:4222')
      expect(result.isErr()).toBe(true)
      expect(result.error.code).toBe('INVALID_FORMAT')
    })

    it('rejects malformed URL', () => {
      const result = NatsUrl.create('not-a-url')
      expect(result.isErr()).toBe(true)
      expect(result.error.code).toBe('INVALID_FORMAT')
    })
  })

  describe('fromTrusted', () => {
    it('parses valid URL without validation', () => {
      const url = NatsUrl.fromTrusted('nats://server:4222')
      expect(url.protocol).toBe('nats')
      expect(url.host).toBe('server')
      expect(url.port).toBe(4222)
    })

    it('handles unparseable URL gracefully', () => {
      const url = NatsUrl.fromTrusted('garbage')
      expect(url.protocol).toBe('nats')
      expect(url.host).toBe('unknown')
      expect(url.port).toBe(4222)
    })
  })

  describe('isSecure', () => {
    it('returns true for tls', () => {
      const url = NatsUrl.fromTrusted('tls://server:4222')
      expect(url.isSecure()).toBe(true)
    })

    it('returns true for wss', () => {
      const url = NatsUrl.fromTrusted('wss://server:443')
      expect(url.isSecure()).toBe(true)
    })

    it('returns false for nats', () => {
      const url = NatsUrl.fromTrusted('nats://server:4222')
      expect(url.isSecure()).toBe(false)
    })

    it('returns false for ws', () => {
      const url = NatsUrl.fromTrusted('ws://server:8080')
      expect(url.isSecure()).toBe(false)
    })
  })

  describe('isWebSocket', () => {
    it('returns true for ws', () => {
      expect(NatsUrl.fromTrusted('ws://server:8080').isWebSocket()).toBe(true)
    })

    it('returns true for wss', () => {
      expect(NatsUrl.fromTrusted('wss://server:443').isWebSocket()).toBe(true)
    })

    it('returns false for nats', () => {
      expect(NatsUrl.fromTrusted('nats://server:4222').isWebSocket()).toBe(false)
    })
  })

  describe('displayString', () => {
    it('returns host:port format', () => {
      const url = NatsUrl.fromTrusted('nats://myserver:4222')
      expect(url.displayString()).toBe('myserver:4222')
    })
  })

  describe('toString', () => {
    it('returns the full URL', () => {
      const url = NatsUrl.fromTrusted('nats://myserver:4222')
      expect(url.toString()).toBe('nats://myserver:4222')
    })
  })
})
