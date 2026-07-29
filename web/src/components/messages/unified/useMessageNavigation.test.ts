import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, act, waitFor } from '@testing-library/react'
import * as messagesApi from '@/api/messages'
import type { MessagesResponse } from '@/api/messages'
import type { Message } from '@/types/nats'
import type { SelectedMessage } from '@/types/messages'
import { useMessageNavigation, type UseMessageNavigationOptions } from './useMessageNavigation'

vi.mock('@/api/messages', () => ({
  getMessages: vi.fn(),
}))

vi.mock('@/utils/toast', () => ({
  toast: { error: vi.fn(), success: vi.fn(), warning: vi.fn() },
}))

const api = vi.mocked(messagesApi)

function msg(sequence: number): Message {
  return {
    sequence,
    subject: `orders.${sequence}`,
    timestamp: sequence * 1000,
    data_base64: '',
    data_size: 0,
    content_type: 'json',
  }
}

function res(seqs: number[]): MessagesResponse {
  return { messages: seqs.map(msg), has_more: false, next_seq: 0 }
}

function selected(sequence: number): SelectedMessage {
  return {
    id: `history-${sequence}`,
    sequence,
    subject: `orders.${sequence}`,
    timestamp: sequence * 1000,
    data_base64: '',
    data_size: 0,
  }
}

function opts(over: Partial<UseMessageNavigationOptions> = {}): UseMessageNavigationOptions {
  return {
    streamName: 'ORDERS',
    connectionId: 'conn-1',
    selectedMessage: selected(10),
    navQuery: { direction: 'backward', subjectFilter: 'orders.*', contentFilter: undefined },
    onSelectMessage: vi.fn(),
    keyboardEnabled: false,
    ...over,
  }
}

describe('useMessageNavigation', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('goNext on a backward list fetches seq-1 backward with the list filters', async () => {
    api.getMessages.mockResolvedValueOnce(res([9]))
    const o = opts()
    const { result } = renderHook(() => useMessageNavigation(o))

    act(() => { result.current.goNext() })

    await waitFor(() => expect(o.onSelectMessage).toHaveBeenCalled())
    expect(api.getMessages).toHaveBeenCalledWith('ORDERS', {
      connection_id: 'conn-1',
      start_seq: 9,
      direction: 'backward',
      limit: 1,
      subject_filter: 'orders.*',
      content_filter: undefined,
    })
    expect(o.onSelectMessage).toHaveBeenCalledWith(
      expect.objectContaining({ id: 'history-9', sequence: 9, isLive: false }),
    )
  })

  it('forward list (jump-to-time): goNext walks newer via forward', async () => {
    api.getMessages.mockResolvedValueOnce(res([11]))
    const o = opts({ navQuery: { direction: 'forward' } })
    const { result } = renderHook(() => useMessageNavigation(o))

    act(() => { result.current.goNext() })

    await waitFor(() => expect(o.onSelectMessage).toHaveBeenCalled())
    expect(api.getMessages).toHaveBeenCalledWith(
      'ORDERS',
      expect.objectContaining({ start_seq: 11, direction: 'forward' }),
    )
  })

  it('empty response marks the direction as edge; edge resets on selection change', async () => {
    api.getMessages.mockResolvedValueOnce(res([]))
    const o = opts()
    const { result, rerender } = renderHook(
      (p: UseMessageNavigationOptions) => useMessageNavigation(p),
      { initialProps: o },
    )

    act(() => { result.current.goNext() })

    await waitFor(() => expect(result.current.nextDisabled).toBe(true))
    expect(o.onSelectMessage).not.toHaveBeenCalled()
    expect(result.current.prevDisabled).toBe(false)

    rerender({ ...o, selectedMessage: selected(11) })
    await waitFor(() => expect(result.current.nextDisabled).toBe(false))
  })

  it('ignores presses while a navigation request is in flight', async () => {
    let resolve!: (v: MessagesResponse) => void
    api.getMessages.mockReturnValueOnce(new Promise((r) => { resolve = r }))
    const o = opts()
    const { result } = renderHook(() => useMessageNavigation(o))

    act(() => { result.current.goNext() })
    act(() => { result.current.goNext() })
    act(() => { result.current.goPrev() })

    await act(async () => { resolve(res([9])) })
    expect(api.getMessages).toHaveBeenCalledTimes(1)
  })

  it('marks edge without an API call when the older step would be start_seq 0', async () => {
    const o = opts({ selectedMessage: selected(1) })
    const { result } = renderHook(() => useMessageNavigation(o))

    act(() => { result.current.goNext() })

    await waitFor(() => expect(result.current.nextDisabled).toBe(true))
    expect(api.getMessages).not.toHaveBeenCalled()
  })

  it('cannot navigate from a message without a sequence (live core_nats)', () => {
    const o = opts({
      selectedMessage: { ...selected(5), id: 'live-abc', sequence: undefined, isLive: true },
    })
    const { result } = renderHook(() => useMessageNavigation(o))

    expect(result.current.canNavigate).toBe(false)
    expect(result.current.prevDisabled).toBe(true)
    expect(result.current.nextDisabled).toBe(true)

    act(() => { result.current.goNext() })
    expect(api.getMessages).not.toHaveBeenCalled()
  })

  it('drops the result when the selection changed while the request was in flight', async () => {
    let resolve!: (v: MessagesResponse) => void
    api.getMessages.mockReturnValueOnce(new Promise((r) => { resolve = r }))
    const o = opts()
    const { result, rerender } = renderHook(
      (p: UseMessageNavigationOptions) => useMessageNavigation(p),
      { initialProps: o },
    )

    act(() => { result.current.goNext() })
    rerender({ ...o, selectedMessage: selected(42) }) // user clicked another row

    await act(async () => { resolve(res([9])) })
    expect(o.onSelectMessage).not.toHaveBeenCalled()
  })

  it('ArrowDown on the document navigates when keyboard is enabled', async () => {
    api.getMessages.mockResolvedValueOnce(res([9]))
    const o = opts({ keyboardEnabled: true })
    renderHook(() => useMessageNavigation(o))

    act(() => {
      document.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowDown', bubbles: true }))
    })

    await waitFor(() => expect(o.onSelectMessage).toHaveBeenCalled())
  })

  it('ignores arrows typed into editable elements', () => {
    const o = opts({ keyboardEnabled: true })
    renderHook(() => useMessageNavigation(o))

    const input = document.createElement('input')
    document.body.appendChild(input)
    input.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowDown', bubbles: true }))

    expect(api.getMessages).not.toHaveBeenCalled()
    input.remove()
  })

  it('does not listen when keyboardEnabled is false', () => {
    const o = opts({ keyboardEnabled: false })
    renderHook(() => useMessageNavigation(o))

    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowDown', bubbles: true }))

    expect(api.getMessages).not.toHaveBeenCalled()
  })
})
