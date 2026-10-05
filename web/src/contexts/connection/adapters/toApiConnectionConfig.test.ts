import { describe, expect, it } from 'vitest'
import { durToMillis } from '@/utils/timestamp'
import { inboxPrefixError, jetStreamTargetErrors, toApiConnectionConfig } from './toApiConnectionConfig'

describe('toApiConnectionConfig', () => {
  it('sends nothing for a new connection without a prefix', () => {
    expect(toApiConnectionConfig(undefined, { inboxPrefix: '  ', jetstreamDomain: '', jetstreamApiPrefix: ' ' })).toBeUndefined()
  })

  it('sets the inbox prefix and keeps the other saved settings', () => {
    const config = toApiConnectionConfig(
      { connectTimeout: 5000, connectionName: 'cli', noEcho: true, noRandomize: false, ignoreDiscoveredServers: true },
      { inboxPrefix: ' _INBOX_alice ', jetstreamDomain: '', jetstreamApiPrefix: '' },
    )
    expect(config?.inboxPrefix).toBe('_INBOX_alice')
    expect(config?.connectionName).toBe('cli')
    expect(config?.noEcho).toBe(true)
    expect(config?.ignoreDiscoveredServers).toBe(true)
    expect(durToMillis(config?.connectTimeout)).toBe(5000)
  })

  it('clears a saved prefix with an empty value', () => {
    const config = toApiConnectionConfig(
      { inboxPrefix: '_INBOX_alice', noEcho: false, noRandomize: false, ignoreDiscoveredServers: false },
      { inboxPrefix: '', jetstreamDomain: '', jetstreamApiPrefix: '' },
    )
    expect(config?.inboxPrefix).toBe('')
  })

  it('sets the JetStream domain or API prefix', () => {
    const config = toApiConnectionConfig(undefined, { inboxPrefix: '', jetstreamDomain: ' hub ', jetstreamApiPrefix: '' })
    expect(config?.jetstreamDomain).toBe('hub')
    expect(config?.jetstreamApiPrefix).toBe('')

    const prefixed = toApiConnectionConfig(undefined, { inboxPrefix: '', jetstreamDomain: '', jetstreamApiPrefix: 'JS.A.API' })
    expect(prefixed?.jetstreamApiPrefix).toBe('JS.A.API')
  })
})

describe('jetStreamTargetErrors', () => {
  it('accepts a domain or a prefix', () => {
    expect(jetStreamTargetErrors('hub', '')).toEqual({})
    expect(jetStreamTargetErrors('', 'JS.A.API')).toEqual({})
  })

  it('names the field that is wrong', () => {
    expect(jetStreamTargetErrors('hub.east', '').domain).toBeDefined()
    expect(jetStreamTargetErrors('', 'JS.*.API').prefix).toBeDefined()
    expect(jetStreamTargetErrors('hub', 'JS.A.API').prefix).toBe('Set a JetStream domain or an API prefix, not both')
  })
})

describe('inboxPrefixError', () => {
  it.each(['', '_INBOX_alice', 'replies.alice'])('accepts %j', (prefix) => {
    expect(inboxPrefixError(prefix)).toBeUndefined()
  })

  it.each(['_INBOX.*', 'replies.>', 'replies.', '.replies', 'my inbox'])('rejects %j', (prefix) => {
    expect(inboxPrefixError(prefix)).toBeDefined()
  })
})
