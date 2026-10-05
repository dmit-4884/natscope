import { describe, expect, it } from 'vitest'
import { durToMillis } from '@/utils/timestamp'
import { inboxPrefixError, toApiConnectionConfig } from './toApiConnectionConfig'

describe('toApiConnectionConfig', () => {
  it('sends nothing for a new connection without a prefix', () => {
    expect(toApiConnectionConfig(undefined, '  ')).toBeUndefined()
  })

  it('sets the inbox prefix and keeps the other saved settings', () => {
    const config = toApiConnectionConfig(
      { connectTimeout: 5000, connectionName: 'cli', noEcho: true, noRandomize: false, ignoreDiscoveredServers: true },
      ' _INBOX_alice ',
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
      '',
    )
    expect(config?.inboxPrefix).toBe('')
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
