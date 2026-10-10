import { afterEach, describe, it, expect, vi } from 'vitest'
import { useLayoutEffect } from 'react'
import { act, render, screen, fireEvent, waitFor } from '@testing-library/react'
import { Code, ConnectError } from '@connectrpc/connect'
import { create, toBinary } from '@bufbuild/protobuf'
import { ErrorInfoSchema } from '@/gen/google/rpc/error_details_pb'
import type { StreamDetail } from '@/types/nats'

const { fixtures } = vi.hoisted(() => ({
  fixtures: {
    defaultViewMode: 'history',
    messagesError: null as unknown,
    messagesData: undefined as unknown,
    messagesEnabled: [] as unknown[],
    liveClients: [] as Array<{ onStats?: (p: { messages_received: number; messages_dropped: number; msg_per_second: number }) => void }>,
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
  useConnectionQuery: (opts: { key: readonly unknown[]; enabled?: boolean }) => {
    if (opts.key[0] === 'messages') {
      fixtures.messagesEnabled.push(opts.enabled)
      return { data: fixtures.messagesData, isLoading: false, isFetching: false, error: fixtures.messagesError, refetch: vi.fn() }
    }
    if (opts.key[0] === 'stream') {
      return { data: fixtures.workQueueStream, isLoading: false, isFetching: false, error: null, refetch: vi.fn() }
    }
    return { data: undefined, isLoading: false, isFetching: false, error: null, refetch: vi.fn() }
  },
}))

vi.mock('@/api/messages', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/api/messages')>()),
  searchMessages: vi.fn(async function* () {
    yield { kind: 'matches', messages: [{ sequence: 41, subject: 'orders.paid', timestamp: 0, data_base64: '', data_size: 0, content_type: 'text' }] }
    yield { kind: 'done', done: { scanned: 120, matched: 1, reason: 'complete', range_first: 1, range_last: 120 } }
  }),
}))

vi.mock('@/contexts/live', async (importOriginal) => {
  class LiveStreamClient {
    subscribe = vi.fn()
    subscribeSubjects = vi.fn()
    unsubscribe = vi.fn()
    disconnect = vi.fn()
    onStats?: (p: { messages_received: number; messages_dropped: number; msg_per_second: number }) => void
    constructor() {
      fixtures.liveClients.push(this)
    }
    connect() {}
  }
  return {
    ...(await importOriginal<typeof import('@/contexts/live')>()),
    LiveStreamClient,
  }
})

vi.mock('@/contexts/settings', () => ({
  useDisplayPreferences: () => ({
    density: 'comfortable',
    defaultViewMode: fixtures.defaultViewMode,
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

import { searchMessages } from '@/api/messages'
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

let scopes = 0

function renderList(scope = { connectionUrl: 'nats://a', streamName: `ORDERS_WORKQUEUE-${++scopes}` }) {
  return render(
    <UnifiedMessageList
      scope={scope}
      streamName="ORDERS_WORKQUEUE"
      connectionId="conn-1"
      onSelectMessage={() => {}}
    />,
  )
}

afterEach(() => {
  fixtures.defaultViewMode = 'history'
  fixtures.messagesData = undefined
})

describe('UnifiedMessageList paging', () => {
  it('offers older messages when a page came back empty but more remain', () => {
    fixtures.messagesError = null
    fixtures.messagesData = { messages: [], has_more: true, next_seq: 4999 }

    renderList()

    expect(screen.getByTestId('load-more')).toBeInTheDocument()
  })
})

describe('UnifiedMessageList realtime skipped count', () => {
  it('shows how many live messages were skipped without blaming the browser', async () => {
    fixtures.messagesError = null
    fixtures.defaultViewMode = 'realtime'
    fixtures.liveClients.length = 0
    renderList()

    await waitFor(() => expect(fixtures.liveClients.length).toBeGreaterThan(0))
    act(() => {
      fixtures.liveClients.forEach((c) => c.onStats?.({ messages_received: 5000, messages_dropped: 4321, msg_per_second: 900 }))
    })

    const skipped = await screen.findByTestId('feed-skipped')
    expect(skipped).toHaveTextContent('4,321 skipped')
    fireEvent.mouseEnter(skipped)
    expect(await screen.findByText(/server could not deliver/i)).toBeInTheDocument()
    expect(screen.queryByText(/browser/i)).not.toBeInTheDocument()
  })
})

describe('UnifiedMessageList view state', () => {
  it('keeps the applied filters when the tab is opened again', async () => {
    fixtures.messagesError = null
    const scope = { connectionUrl: 'nats://a', streamName: 'KEEP' }
    const first = renderList(scope)
    fireEvent.click(screen.getByRole('button', { name: /filters/i }))
    fireEvent.change(screen.getByLabelText(/payload search/i), { target: { value: 'needle' } })
    fireEvent.click(screen.getByRole('button', { name: 'Apply' }))
    first.unmount()

    renderList(scope)

    await waitFor(() => expect(screen.getByTestId('search-status')).toBeInTheDocument())
  })

  it('opens in the default view mode from its first frame', () => {
    fixtures.messagesError = null
    fixtures.defaultViewMode = 'realtime'
    const frames: string[] = []
    function Frame() {
      useLayoutEffect(() => {
        frames.push(document.body.textContent ?? '')
      })
      return null
    }

    render(
      <>
        <UnifiedMessageList
          scope={{ connectionUrl: 'nats://a', streamName: 'LIVE' }}
          streamName="ORDERS_WORKQUEUE"
          connectionId="conn-1"
          onSelectMessage={() => {}}
        />
        <Frame />
      </>,
    )

    expect(frames[0]).toContain('Clear')
  })
})

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

describe('UnifiedMessageList search', () => {
  it('searches the whole stream on the server once a text filter is applied', async () => {
    fixtures.messagesError = null
    renderList()

    fireEvent.click(screen.getByRole('button', { name: /filters/i }))
    fireEvent.change(screen.getByLabelText(/payload search/i), { target: { value: 'needle' } })
    fireEvent.click(screen.getByRole('button', { name: 'Apply' }))

    await waitFor(() => expect(screen.getByTestId('search-status')).toHaveTextContent('120 messages read · 1 found'))
    expect(vi.mocked(searchMessages)).toHaveBeenCalledWith(
      'ORDERS_WORKQUEUE',
      expect.objectContaining({ connection_id: 'conn-1', text: 'needle', direction: 'backward' }),
      expect.any(AbortSignal),
    )
    expect(screen.queryByTestId('search-empty')).not.toBeInTheDocument()
  })
})

describe('UnifiedMessageList search states', () => {
  const applySearch = () => {
    fireEvent.click(screen.getByRole('button', { name: /filters/i }))
    fireEvent.change(screen.getByLabelText(/payload search/i), { target: { value: 'needle' } })
    fireEvent.click(screen.getByRole('button', { name: 'Apply' }))
  }

  it('stops reading the page as soon as a search is applied', () => {
    fixtures.messagesError = null
    renderList()
    fixtures.messagesEnabled = []

    applySearch()

    expect(fixtures.messagesEnabled[fixtures.messagesEnabled.length - 1]).toBe(false)
  })

  it('hides a list error from before the search', async () => {
    fixtures.messagesError = streamNotFoundError()
    renderList()

    applySearch()

    await waitFor(() => expect(screen.getByTestId('search-status')).toBeInTheDocument())
    expect(screen.queryByText('Stream not found')).not.toBeInTheDocument()
  })

  it('says plainly that nothing matched once the whole range was searched', async () => {
    fixtures.messagesError = null
    vi.mocked(searchMessages).mockImplementationOnce(async function* () {
      yield { kind: 'done', done: { scanned: 120, matched: 0, reason: 'complete', range_first: 1, range_last: 120 } }
    })
    renderList()

    applySearch()

    await waitFor(() => expect(screen.getByTestId('search-empty')).toHaveTextContent(/^No message matched\.$/))
  })
})
