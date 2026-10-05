import { describe, expect, it } from 'vitest'
import { isJetStreamControlReply } from './replySubject'

describe('isJetStreamControlReply', () => {
  it.each(['$JS.ACK.ORDERS.c1.1.5.5.1700000000000000000.0', '$JS.FC.ORDERS.abc.1'])('is true for %s', (reply) => {
    expect(isJetStreamControlReply(reply)).toBe(true)
  })

  it.each(['_INBOX.abc.1', 'my.reply.inbox', '$JS.API.STREAM.INFO.X', '$JS.ACKS.x'])('is false for %s', (reply) => {
    expect(isJetStreamControlReply(reply)).toBe(false)
  })
})
