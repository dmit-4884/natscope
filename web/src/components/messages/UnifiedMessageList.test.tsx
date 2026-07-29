import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { Code, ConnectError } from '@connectrpc/connect'
import { create, toBinary } from '@bufbuild/protobuf'
import { ErrorInfoSchema } from '@/gen/google/rpc/error_details_pb'
import type { StreamDetail } from '@/types/nats'

const { fixtures } = vi.hoisted(() => ({
  fixtures: {
    messagesError: null as unknown,
    workQueueStream: {
      name: 'ORDERS_WORKQUEUE',
      subjects: ['orders.>'],
      messages: 0,
      bytes: 0,
      consumer_count: 0,
      created: 0,
      config: { retention: 'workqueue', max_msgs: -1, max_bytes: -1, max_age: 0 },
      state: { messages: 0, bytes: 0, first_seq: 0, last_seq: 0, first_ts: 0, last_ts: 0 },
      consumers: [],
    } satisfies StreamDetail,
  },
}))

vi.mock('@tanstack/react-query', () => ({
  useQueryClient: () => ({ invalidateQueries: vi.fn() }),
}))

vi.mock('@/hooks/useConnectionQuery', () => ({
  CONNECTION_QUERY_PREFIX: 'conn',
  useConnectionQuery: (opts: { key: readonly unknown[] }) => {
    if (opts.key[0] === 'messages') {
      return { data: undefined, isLoading: false, isFetching: false, error: fixtures.messagesError, refetch: vi.fn() }
    }
    if (opts.key[0] === 'stream') {
      return { data: fixtures.workQueueStream, isLoading: false, isFetching: false, error: null, refetch: vi.fn() }
    }
    return { data: undefined, isLoading: false, isFetching: false, error: null, refetch: vi.fn() }
  },
}))

vi.mock('@/contexts/settings', () => ({
  useDisplayPreferences: () => ({
    density: 'comfortable',
    defaultViewMode: 'history',
    timestampFormat: 'relative',
    jsonIndentSize: 2,
    autoScrollLive: true,
  }),
  useMessagesPolicy: () => ({
    fetchMethod: 'consumer',
    defaultPageSize: 50,
    defaultDirection: 'backward',
    maxPayloadBytesInList: 65536,
    defaultExportFormat: 'json',
    exportRangeLimit: 50000,
  }),
  useLivePolicy: () => ({ subscriptionMode: 'core_nats', maxDisplayRate: 0 }),
  useUpdateSettings: () => ({ mutate: vi.fn() }),
  useSettings: () => ({ isSuccess: true, data: undefined }),
}))

import UnifiedMessageList from './UnifiedMessageList'

function workQueueConsumerError(): ConnectError {
  const err = new ConnectError('cannot create read consumer on WorkQueue stream', Code.FailedPrecondition)
  err.details = [{
    type: ErrorInfoSchema.typeName,
    value: toBinary(ErrorInfoSchema, create(ErrorInfoSchema, { reason: 'NATS_WORKQUEUE_CONSUMER_NOT_ALLOWED' })),
  }]
  return err
}

function streamNotFoundError(): ConnectError {
  const err = new ConnectError('stream not found', Code.NotFound)
  err.details = [{
    type: ErrorInfoSchema.typeName,
    value: toBinary(ErrorInfoSchema, create(ErrorInfoSchema, { reason: 'NATS_STREAM_NOT_FOUND' })),
  }]
  return err
}

function renderList() {
  return render(
    <UnifiedMessageList
      streamName="ORDERS_WORKQUEUE"
      connectionId="conn-1"
      onSelectMessage={() => {}}
    />,
  )
}

describe('UnifiedMessageList WorkQueue history error banner', () => {
  it('suppresses the red alert for the expected WorkQueue consumer rejection and shows only the amber warning', () => {
    fixtures.messagesError = workQueueConsumerError()
    renderList()

    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
    expect(screen.getByText(/WorkQueue stream:/)).toBeInTheDocument()
  })

  it('still shows the red alert for an unrelated history error', () => {
    fixtures.messagesError = streamNotFoundError()
    renderList()

    expect(screen.getByRole('alert')).toBeInTheDocument()
    expect(screen.getByText('Stream not found')).toBeInTheDocument()
    expect(screen.getByText(/WorkQueue stream:/)).toBeInTheDocument()
  })
})
