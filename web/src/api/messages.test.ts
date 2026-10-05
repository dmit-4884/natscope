import { describe, it, expect, vi, beforeEach } from 'vitest'
import { create } from '@bufbuild/protobuf'
import {
  Direction,
  GetNextMessageResponseSchema,
  SearchMessagesResponseSchema,
  SearchStopReason,
} from '../gen/services/grpc/nats/v1/messages/nats_messages_service_pb'

const getNextMessageMock = vi.hoisted(() => vi.fn())
const searchMessagesMock = vi.hoisted(() => vi.fn())

vi.mock('./grpc/clients', () => ({
  messagesClient: { getNextMessage: getNextMessageMock, searchMessages: searchMessagesMock },
}))

import { getNextMessage, searchMessages, type SearchEvent } from './messages'

describe('getNextMessage', () => {
  beforeEach(() => getNextMessageMock.mockReset())

  it('asks for the first match at or after the sequence and maps it', async () => {
    getNextMessageMock.mockResolvedValue(
      create(GetNextMessageResponseSchema, {
        message: { sequence: 7n, subject: 'orders.paid', dataBase64: 'e30=', dataSize: 2, contentType: 'json' },
      }),
    )

    const message = await getNextMessage('conn-1', 'ORDERS', 5, ['orders.paid', 'orders.*'])

    expect(getNextMessageMock).toHaveBeenCalledWith(
      { connectionId: 'conn-1', streamName: 'ORDERS', startSeq: 5n, subjects: ['orders.paid', 'orders.*'] },
      { signal: undefined },
    )
    expect(message).toMatchObject({ sequence: 7, subject: 'orders.paid', content_type: 'json' })
  })

  it('is null when nothing matches', async () => {
    getNextMessageMock.mockResolvedValue(create(GetNextMessageResponseSchema, {}))

    expect(await getNextMessage('conn-1', 'ORDERS', 99, [])).toBeNull()
  })
})

describe('searchMessages', () => {
  beforeEach(() => {
    searchMessagesMock.mockReset()
  })

  it('sends the filters and maps every event of the stream', async () => {
    async function* events() {
      yield create(SearchMessagesResponseSchema, {
        event: { case: 'progress', value: { scanned: 10n, matched: 1n, currentSeq: 10n, rangeFirstSeq: 1n, rangeLastSeq: 900n } },
      })
      yield create(SearchMessagesResponseSchema, {
        event: { case: 'matches', value: { messages: [{ sequence: 7n, subject: 'orders.paid', dataBase64: 'e30=', contentType: 'json' }] } },
      })
      yield create(SearchMessagesResponseSchema, {
        event: { case: 'done', value: { scanned: 100n, matched: 1n, reason: SearchStopReason.SCAN_LIMIT, rangeFirstSeq: 1n, rangeLastSeq: 900n, nextSeq: 101n } },
      })
    }
    searchMessagesMock.mockReturnValue(events())

    const seen: SearchEvent[] = []
    for await (const event of searchMessages('ORDERS', {
      connection_id: 'conn-1',
      subject_filter: 'orders.>',
      direction: 'backward',
      from_seq: 5,
      to_time: Date.UTC(2026, 9, 5),
      cursor_seq: 200,
      text: 'needle',
      regex: true,
      header_name: 'X-Trace',
      header_value: 'abc',
    })) {
      seen.push(event)
    }

    const [request] = searchMessagesMock.mock.calls[0]
    expect(request).toMatchObject({
      connectionId: 'conn-1',
      streamName: 'ORDERS',
      subjectFilter: 'orders.>',
      direction: Direction.BACKWARD,
      fromSeq: 5n,
      cursorSeq: 200n,
      text: 'needle',
      regex: true,
      headerName: 'X-Trace',
      headerValue: 'abc',
    })
    expect(request.toTime.seconds).toBe(BigInt(Date.UTC(2026, 9, 5) / 1000))
    expect(seen).toEqual([
      { kind: 'progress', progress: { scanned: 10, matched: 1, current_seq: 10, range_first: 1, range_last: 900 } },
      { kind: 'matches', messages: [expect.objectContaining({ sequence: 7, subject: 'orders.paid' })] },
      { kind: 'done', done: { scanned: 100, matched: 1, reason: 'scan_limit', range_first: 1, range_last: 900, next_seq: 101 } },
    ])
  })
})
