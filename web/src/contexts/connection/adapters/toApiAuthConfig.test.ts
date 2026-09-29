import { describe, it, expect } from 'vitest'
import { AuthMethod as ProtoAuthMethod } from '@/gen/types/nats/nats_connection_pb'
import { AuthConfig } from '../domain/value-objects/AuthConfig'
import { toApiAuthConfig } from './toApiAuthConfig'

describe('toApiAuthConfig', () => {
  it('returns undefined for none when not explicit (leave auth untouched)', () => {
    const result = toApiAuthConfig(AuthConfig.fromTrusted({ method: 'none' }))
    expect(result).toBeUndefined()
  })

  it('returns undefined for a null/undefined config when not explicit', () => {
    expect(toApiAuthConfig(undefined)).toBeUndefined()
    expect(toApiAuthConfig(null)).toBeUndefined()
  })

  it('emits AUTH_METHOD_UNSPECIFIED for none when explicit, never undefined', () => {
    const result = toApiAuthConfig(AuthConfig.fromTrusted({ method: 'none' }), { explicit: true })
    expect(result).toBeDefined()
    expect(result?.method).toBe(ProtoAuthMethod.UNSPECIFIED)
  })

  it('emits AUTH_METHOD_UNSPECIFIED for a null config when explicit', () => {
    const result = toApiAuthConfig(undefined, { explicit: true })
    expect(result).toBeDefined()
    expect(result?.method).toBe(ProtoAuthMethod.UNSPECIFIED)
  })

  it('carries method and material through for a populated config regardless of explicit', () => {
    const config = AuthConfig.fromTrusted({ method: 'token', token: 'secret-token' })
    const result = toApiAuthConfig(config, { explicit: true })
    expect(result?.method).toBe(ProtoAuthMethod.TOKEN)
    expect(result?.token).toBe('secret-token')
  })
})
