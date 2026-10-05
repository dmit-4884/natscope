import { describe, it, expect, vi, beforeEach } from 'vitest'
import { act, renderHook, waitFor } from '@testing-library/react'
import * as api from '@/api/messages'
import type { SearchEvent, SearchParams } from '@/api/messages'
import type { Message } from '@/types/nats'
import { useMessageSearch, type SearchQuery } from './useMessageSearch'

vi.mock('@/api/messages', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/api/messages')>()),
  searchMessages: vi.fn(),
}))

const mockedSearch = vi.mocked(api.searchMessages)

const msg = (sequence: number): Message => ({
  sequence,
  subject: 'orders.paid',
  timestamp: 0,
  data_base64: '',
  data_size: 0,
  content_type: 'text',
})

function scripted(events: SearchEvent[], hold?: Promise<void>) {
  return async function* (_stream: string, _params: SearchParams, signal?: AbortSignal) {
    for (const event of events) {
      yield event
    }
    if (hold) {
      await new Promise<void>((resolve, reject) => {
        void hold.then(resolve)
        signal?.addEventListener('abort', () => reject(new DOMException('aborted', 'AbortError')))
      })
    }
  }
}

const progress = (current: number, scanned: number, resume?: number): SearchEvent => ({
  kind: 'progress',
  progress: { scanned, matched: 0, current_seq: current, range_first: 1, range_last: 1000, resume_seq: resume },
})
const done = (next?: number): SearchEvent => ({
  kind: 'done',
  done: { scanned: 100, matched: 1, reason: next ? 'scan_limit' : 'complete', range_first: 1, range_last: 1000, next_seq: next },
})

const query: SearchQuery = { direction: 'backward', text: 'needle' }

describe('useMessageSearch', () => {
  beforeEach(() => {
    mockedSearch.mockReset()
  })

  it('stays idle without a query', () => {
    const { result } = renderHook(() => useMessageSearch('conn-1', 'ORDERS', null))
    expect(result.current.status).toBe('idle')
    expect(mockedSearch).not.toHaveBeenCalled()
  })

  it('collects matches and ends with the summary', async () => {
    mockedSearch.mockImplementation(scripted([progress(900, 100), { kind: 'matches', messages: [msg(950)] }, done(899)]))
    const { result } = renderHook(() => useMessageSearch('conn-1', 'ORDERS', query))

    await waitFor(() => expect(result.current.status).toBe('done'))
    expect(result.current.messages.map((m) => m.sequence)).toEqual([950])
    expect(result.current.done?.next_seq).toBe(899)
    expect(result.current.canContinue).toBe(true)
    expect(mockedSearch).toHaveBeenCalledWith('ORDERS', expect.objectContaining({ connection_id: 'conn-1', text: 'needle', direction: 'backward' }), expect.any(AbortSignal))
  })

  it('continues where the run stopped and keeps the earlier matches', async () => {
    mockedSearch.mockImplementationOnce(scripted([{ kind: 'matches', messages: [msg(950)] }, done(899)]))
    mockedSearch.mockImplementationOnce(scripted([{ kind: 'matches', messages: [msg(10)] }, done()]))
    const { result } = renderHook(() => useMessageSearch('conn-1', 'ORDERS', query))
    await waitFor(() => expect(result.current.status).toBe('done'))

    act(() => result.current.more())
    await waitFor(() => expect(result.current.messages.map((m) => m.sequence)).toEqual([950, 10]))
    expect(mockedSearch.mock.calls[1][1]).toMatchObject({ cursor_seq: 899 })
    expect(result.current.canContinue).toBe(false)
  })

  it('stops on request and picks up where the server says nothing is skipped', async () => {
    let release = () => {}
    mockedSearch.mockImplementationOnce(scripted([progress(700, 300, 999)], new Promise<void>((r) => (release = r))))
    const { result } = renderHook(() => useMessageSearch('conn-1', 'ORDERS', query))
    await waitFor(() => expect(result.current.progress?.current_seq).toBe(700))

    act(() => result.current.stop())
    expect(result.current.status).toBe('stopped')
    const signal = mockedSearch.mock.calls[0][2]!
    expect(signal.aborted).toBe(true)
    release()

    mockedSearch.mockImplementationOnce(scripted([done()]))
    act(() => result.current.more())
    await waitFor(() => expect(result.current.status).toBe('done'))
    expect(mockedSearch.mock.calls[1][1]).toMatchObject({ cursor_seq: 999 })
  })

  it('keeps its place when stopped right after searching further', async () => {
    mockedSearch.mockImplementationOnce(scripted([done(899)]))
    const { result } = renderHook(() => useMessageSearch('conn-1', 'ORDERS', query))
    await waitFor(() => expect(result.current.status).toBe('done'))

    mockedSearch.mockImplementationOnce(scripted([], new Promise<void>(() => {})))
    act(() => result.current.more())
    act(() => result.current.stop())

    expect(result.current.status).toBe('stopped')
    expect(result.current.canContinue).toBe(true)
    mockedSearch.mockImplementationOnce(scripted([done()]))
    act(() => result.current.more())
    await waitFor(() => expect(mockedSearch).toHaveBeenCalledTimes(3))
    expect(mockedSearch.mock.calls[2][1]).toMatchObject({ cursor_seq: 899 })
  })

  it('shows a match once even when a run sends it again', async () => {
    mockedSearch.mockImplementationOnce(scripted([{ kind: 'matches', messages: [msg(101), msg(150)] }, done(151)]))
    mockedSearch.mockImplementationOnce(scripted([{ kind: 'matches', messages: [msg(150), msg(200)] }, done()]))
    const { result } = renderHook(() => useMessageSearch('conn-1', 'ORDERS', { ...query, direction: 'forward' }))
    await waitFor(() => expect(result.current.status).toBe('done'))

    act(() => result.current.more())
    await waitFor(() => expect(result.current.done?.next_seq).toBeUndefined())
    expect(result.current.messages.map((m) => m.sequence)).toEqual([101, 150, 200])
  })

  it('starts over when the query changes', async () => {
    mockedSearch.mockImplementation(scripted([{ kind: 'matches', messages: [msg(950)] }, done()]))
    const { result, rerender } = renderHook(({ q }) => useMessageSearch('conn-1', 'ORDERS', q), { initialProps: { q: query } })
    await waitFor(() => expect(result.current.status).toBe('done'))

    mockedSearch.mockImplementation(scripted([{ kind: 'matches', messages: [msg(5)] }, done()]))
    rerender({ q: { ...query, text: 'other' } })
    await waitFor(() => expect(result.current.messages.map((m) => m.sequence)).toEqual([5]))
  })

  it('reports a failed search', async () => {
    mockedSearch.mockImplementation(async function* () {
      yield progress(1, 1)
      throw new Error('boom')
    })
    const { result } = renderHook(() => useMessageSearch('conn-1', 'ORDERS', query))
    await waitFor(() => expect(result.current.status).toBe('error'))
    expect((result.current.error as Error).message).toBe('boom')
  })

  it('goes on from where a failed run got to, keeping its matches', async () => {
    mockedSearch.mockImplementationOnce(async function* () {
      yield { kind: 'matches', messages: [msg(950)] } satisfies SearchEvent
      yield progress(800, 200, 799)
      throw new Error('boom')
    })
    const { result } = renderHook(() => useMessageSearch('conn-1', 'ORDERS', query))
    await waitFor(() => expect(result.current.status).toBe('error'))
    expect(result.current.canContinue).toBe(true)

    mockedSearch.mockImplementationOnce(scripted([{ kind: 'matches', messages: [msg(10)] }, done()]))
    act(() => result.current.more())
    await waitFor(() => expect(result.current.status).toBe('done'))
    expect(mockedSearch.mock.calls[1][1]).toMatchObject({ cursor_seq: 799 })
    expect(result.current.messages.map((m) => m.sequence)).toEqual([950, 10])
  })
})

describe('useMessageSearch totals', () => {
  beforeEach(() => {
    mockedSearch.mockReset()
  })

  it('adds up what every run read and keeps the range between runs', async () => {
    mockedSearch.mockImplementationOnce(scripted([progress(900, 100), done(899)]))
    mockedSearch.mockImplementationOnce(scripted([progress(400, 50), done()]))
    const { result } = renderHook(() => useMessageSearch('conn-1', 'ORDERS', query))
    await waitFor(() => expect(result.current.status).toBe('done'))
    expect(result.current.scanned).toBe(100)
    expect(result.current.range).toEqual({ first: 1, last: 1000 })

    act(() => result.current.more())
    await waitFor(() => expect(result.current.done?.next_seq).toBeUndefined())
    expect(result.current.scanned).toBe(200)
    expect(result.current.range).toEqual({ first: 1, last: 1000 })
  })

  it('restarts from the beginning', async () => {
    mockedSearch.mockImplementation(scripted([{ kind: 'matches', messages: [msg(950)] }, done(899)]))
    const { result } = renderHook(() => useMessageSearch('conn-1', 'ORDERS', query))
    await waitFor(() => expect(result.current.status).toBe('done'))

    act(() => result.current.restart())
    await waitFor(() => expect(mockedSearch).toHaveBeenCalledTimes(2))
    await waitFor(() => expect(result.current.status).toBe('done'))
    expect(result.current.messages.map((m) => m.sequence)).toEqual([950])
    expect(mockedSearch.mock.calls[1][1].cursor_seq).toBeUndefined()
  })
})
