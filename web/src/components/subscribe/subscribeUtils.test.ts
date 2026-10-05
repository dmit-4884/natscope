import { describe, it, expect } from 'vitest'
import type { LiveMessage } from '../messages/unified/messageListUtils'
import { filterReceived, publishSubjectError, subscribeSubjectError } from './subscribeUtils'

describe('subscribeSubjectError', () => {
  it('accepts literal subjects and whole-token wildcards', () => {
    for (const s of ['orders', 'orders.*', 'orders.>', '>', '*.created', '$JS.EVENT.>']) {
      expect(subscribeSubjectError(s), s).toBeNull()
    }
  })

  it('explains what is wrong with a malformed pattern', () => {
    expect(subscribeSubjectError('')).toBe('Enter a subject')
    expect(subscribeSubjectError('orders created')).toBe('A subject cannot contain spaces')
    expect(subscribeSubjectError('orders..x')).toBe('A subject cannot have empty tokens')
    expect(subscribeSubjectError('orders.>.x')).toBe('> must be the last token')
    expect(subscribeSubjectError('orders.ab*')).toBe('Wildcards must fill a whole token, like orders.* or orders.>')
  })
})

describe('publishSubjectError', () => {
  it('rejects wildcards', () => {
    expect(publishSubjectError('orders.new')).toBeNull()
    expect(publishSubjectError('orders.*')).toBe('Publish to a literal subject, without * or > wildcards')
  })
})


describe('filterReceived', () => {
  const msg = (subject: string, text: string): LiveMessage => ({
    id: subject,
    stream_name: '',
    subject,
    timestamp: 0,
    data_base64: btoa(text),
    data_size: text.length,
  })
  const messages = [msg('orders.new', '{"id":42}'), msg('audit.login', 'user=bob')]

  it('matches the subject or the payload text, case-insensitively', () => {
    expect(filterReceived(messages, 'ORDERS').map((m) => m.subject)).toEqual(['orders.new'])
    expect(filterReceived(messages, 'bob').map((m) => m.subject)).toEqual(['audit.login'])
    expect(filterReceived(messages, '')).toBe(messages)
  })

  it('leaves out muted subjects', () => {
    expect(filterReceived(messages, '', null, ['audit.>']).map((m) => m.subject)).toEqual(['orders.new'])
    expect(filterReceived(messages, 'bob', null, ['audit.login'])).toEqual([])
  })

  it('narrows to one subject', () => {
    expect(filterReceived(messages, '', 'audit.login').map((m) => m.subject)).toEqual(['audit.login'])
  })
})
