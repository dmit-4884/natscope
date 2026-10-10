import { beforeEach, describe, expect, it, vi } from 'vitest'

const toastSpy = vi.hoisted(() => ({ warning: vi.fn(), error: vi.fn() }))

vi.mock('@/utils/toast', () => ({ toast: toastSpy }))

type Store = typeof import('./bookmarkStore')['useBookmarkStore']

function bookmark(sequence: number, note?: string) {
  return {
    id: `id-${sequence}`,
    connectionId: 'c1',
    streamName: 'S',
    sequence,
    subject: 's',
    createdAt: new Date(sequence).toISOString(),
    note,
  }
}

function fill(count: number, withNote: (seq: number) => boolean): ReturnType<typeof bookmark>[] {
  return Array.from({ length: count }, (_, i) => bookmark(i + 1, withNote(i + 1) ? 'keep me' : undefined))
}

const added = { connectionId: 'c1', streamName: 'S', sequence: 9999, subject: 's' }

describe('bookmark cap', () => {
  let store: Store

  beforeEach(async () => {
    const data = new Map<string, string>()
    vi.stubGlobal('localStorage', {
      getItem: (key: string) => data.get(key) ?? null,
      setItem: (key: string, value: string) => void data.set(key, value),
      removeItem: (key: string) => void data.delete(key),
    })
    vi.resetModules()
    toastSpy.warning.mockClear()
    toastSpy.error.mockClear()
    store = (await import('./bookmarkStore')).useBookmarkStore
  })

  it('evicts the oldest bookmark without a note and tells the user once', () => {
    store.setState({ bookmarks: fill(500, (seq) => seq === 1) })

    store.getState().addBookmark(added)

    const bookmarks = store.getState().bookmarks
    expect(bookmarks).toHaveLength(500)
    expect(bookmarks.some((b) => b.sequence === 1)).toBe(true)
    expect(bookmarks.some((b) => b.sequence === 2)).toBe(false)
    expect(bookmarks.some((b) => b.sequence === 9999)).toBe(true)
    expect(toastSpy.warning).toHaveBeenCalledTimes(1)
    expect(toastSpy.warning.mock.calls[0][0]).toMatch(/500/)

    store.getState().addBookmark({ ...added, sequence: 10_000 })
    expect(toastSpy.warning).toHaveBeenCalledTimes(1)
  })

  it('never evicts a bookmark that has a note: refuses the new one and says why', () => {
    store.setState({ bookmarks: fill(500, () => true) })

    store.getState().addBookmark(added)

    const bookmarks = store.getState().bookmarks
    expect(bookmarks).toHaveLength(500)
    expect(bookmarks.some((b) => b.sequence === 9999)).toBe(false)
    expect(toastSpy.error).toHaveBeenCalledTimes(1)
    expect(toastSpy.error.mock.calls[0][0]).toMatch(/note/i)
  })

  it('stays silent below the cap', () => {
    store.setState({ bookmarks: fill(10, () => false) })

    store.getState().addBookmark(added)

    expect(store.getState().bookmarks).toHaveLength(11)
    expect(toastSpy.warning).not.toHaveBeenCalled()
    expect(toastSpy.error).not.toHaveBeenCalled()
  })
})
