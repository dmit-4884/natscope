import { describe, it, expect, beforeEach } from 'vitest'
import { create } from 'zustand'
import type { Bookmark } from './bookmarkStore'

// Non-persisted store for testing
const useTestStore = create<{
  bookmarks: Bookmark[]
  addBookmark: (bookmark: Omit<Bookmark, 'id' | 'createdAt'>) => void
  removeBookmark: (id: string) => void
  updateNote: (id: string, note: string) => void
  isBookmarked: (connectionId: string, streamName: string, sequence: number) => boolean
  getBookmark: (connectionId: string, streamName: string, sequence: number) => Bookmark | undefined
  getStreamBookmarks: (connectionId: string, streamName: string) => Bookmark[]
  getConnectionBookmarks: (connectionId: string) => Bookmark[]
  clearConnectionBookmarks: (connectionId: string) => void
}>((set, get) => ({
  bookmarks: [],

  addBookmark: (bookmarkData) => {
    const { bookmarks } = get()
    const newBookmark: Bookmark = {
      ...bookmarkData,
      id: crypto.randomUUID(),
      createdAt: new Date().toISOString(),
    }
    set({ bookmarks: [...bookmarks, newBookmark] })
  },

  removeBookmark: (id) => {
    const { bookmarks } = get()
    set({ bookmarks: bookmarks.filter((b) => b.id !== id) })
  },

  updateNote: (id, note) => {
    const { bookmarks } = get()
    set({
      bookmarks: bookmarks.map((b) =>
        b.id === id ? { ...b, note } : b
      ),
    })
  },

  isBookmarked: (connectionId, streamName, sequence) => {
    const { bookmarks } = get()
    return bookmarks.some(
      (b) => b.connectionId === connectionId && b.streamName === streamName && b.sequence === sequence
    )
  },

  getBookmark: (connectionId, streamName, sequence) => {
    const { bookmarks } = get()
    return bookmarks.find(
      (b) => b.connectionId === connectionId && b.streamName === streamName && b.sequence === sequence
    )
  },

  getStreamBookmarks: (connectionId, streamName) => {
    const { bookmarks } = get()
    return bookmarks
      .filter((b) => b.connectionId === connectionId && b.streamName === streamName)
      .sort((a, b) => b.sequence - a.sequence)
  },

  getConnectionBookmarks: (connectionId) => {
    const { bookmarks } = get()
    return bookmarks.filter((b) => b.connectionId === connectionId)
  },

  clearConnectionBookmarks: (connectionId) => {
    const { bookmarks } = get()
    set({ bookmarks: bookmarks.filter((b) => b.connectionId !== connectionId) })
  },
}))

describe('bookmarkStore', () => {
  beforeEach(() => {
    useTestStore.setState({ bookmarks: [] })
  })

  it('adds a bookmark', () => {
    useTestStore.getState().addBookmark({
      connectionId: 'conn1',
      streamName: 'ORDERS',
      sequence: 42,
      subject: 'orders.created',
    })

    const bookmarks = useTestStore.getState().bookmarks
    expect(bookmarks).toHaveLength(1)
    expect(bookmarks[0].connectionId).toBe('conn1')
    expect(bookmarks[0].streamName).toBe('ORDERS')
    expect(bookmarks[0].sequence).toBe(42)
    expect(bookmarks[0].subject).toBe('orders.created')
    expect(bookmarks[0].id).toBeDefined()
    expect(bookmarks[0].createdAt).toBeDefined()
  })

  it('removes a bookmark', () => {
    useTestStore.getState().addBookmark({
      connectionId: 'conn1',
      streamName: 'ORDERS',
      sequence: 1,
      subject: 's1',
    })
    useTestStore.getState().addBookmark({
      connectionId: 'conn1',
      streamName: 'ORDERS',
      sequence: 2,
      subject: 's2',
    })

    const id = useTestStore.getState().bookmarks[0].id
    useTestStore.getState().removeBookmark(id)
    expect(useTestStore.getState().bookmarks).toHaveLength(1)
  })

  it('updates note', () => {
    useTestStore.getState().addBookmark({
      connectionId: 'conn1',
      streamName: 'ORDERS',
      sequence: 1,
      subject: 's',
    })

    const id = useTestStore.getState().bookmarks[0].id
    useTestStore.getState().updateNote(id, 'important message')
    expect(useTestStore.getState().bookmarks[0].note).toBe('important message')
  })

  it('checks isBookmarked', () => {
    useTestStore.getState().addBookmark({
      connectionId: 'conn1',
      streamName: 'ORDERS',
      sequence: 42,
      subject: 's',
    })

    expect(useTestStore.getState().isBookmarked('conn1', 'ORDERS', 42)).toBe(true)
    expect(useTestStore.getState().isBookmarked('conn1', 'ORDERS', 43)).toBe(false)
    expect(useTestStore.getState().isBookmarked('conn2', 'ORDERS', 42)).toBe(false)
  })

  it('gets bookmark by identity', () => {
    useTestStore.getState().addBookmark({
      connectionId: 'conn1',
      streamName: 'ORDERS',
      sequence: 42,
      subject: 's',
    })

    const bookmark = useTestStore.getState().getBookmark('conn1', 'ORDERS', 42)
    expect(bookmark).toBeDefined()
    expect(bookmark!.sequence).toBe(42)

    expect(useTestStore.getState().getBookmark('conn1', 'ORDERS', 999)).toBeUndefined()
  })

  it('gets stream bookmarks sorted by sequence desc', () => {
    useTestStore.getState().addBookmark({ connectionId: 'conn1', streamName: 'ORDERS', sequence: 1, subject: 's' })
    useTestStore.getState().addBookmark({ connectionId: 'conn1', streamName: 'ORDERS', sequence: 3, subject: 's' })
    useTestStore.getState().addBookmark({ connectionId: 'conn1', streamName: 'ORDERS', sequence: 2, subject: 's' })
    useTestStore.getState().addBookmark({ connectionId: 'conn1', streamName: 'USERS', sequence: 1, subject: 's' })

    const bookmarks = useTestStore.getState().getStreamBookmarks('conn1', 'ORDERS')
    expect(bookmarks).toHaveLength(3)
    expect(bookmarks[0].sequence).toBe(3)
    expect(bookmarks[1].sequence).toBe(2)
    expect(bookmarks[2].sequence).toBe(1)
  })

  it('gets connection bookmarks', () => {
    useTestStore.getState().addBookmark({ connectionId: 'conn1', streamName: 'ORDERS', sequence: 1, subject: 's' })
    useTestStore.getState().addBookmark({ connectionId: 'conn1', streamName: 'USERS', sequence: 1, subject: 's' })
    useTestStore.getState().addBookmark({ connectionId: 'conn2', streamName: 'ORDERS', sequence: 1, subject: 's' })

    expect(useTestStore.getState().getConnectionBookmarks('conn1')).toHaveLength(2)
    expect(useTestStore.getState().getConnectionBookmarks('conn2')).toHaveLength(1)
  })

  it('clears only the given connection bookmarks', () => {
    useTestStore.getState().addBookmark({ connectionId: 'conn1', streamName: 'ORDERS', sequence: 1, subject: 's' })
    useTestStore.getState().addBookmark({ connectionId: 'conn1', streamName: 'ORDERS', sequence: 2, subject: 's' })
    useTestStore.getState().addBookmark({ connectionId: 'conn2', streamName: 'ORDERS', sequence: 1, subject: 's' })

    useTestStore.getState().clearConnectionBookmarks('conn1')
    expect(useTestStore.getState().bookmarks).toHaveLength(1)
    expect(useTestStore.getState().bookmarks[0].connectionId).toBe('conn2')
  })

  it('generates unique IDs', () => {
    useTestStore.getState().addBookmark({ connectionId: 'conn1', streamName: 'ORDERS', sequence: 1, subject: 's' })
    useTestStore.getState().addBookmark({ connectionId: 'conn1', streamName: 'ORDERS', sequence: 2, subject: 's' })

    const ids = useTestStore.getState().bookmarks.map((b) => b.id)
    expect(ids[0]).not.toBe(ids[1])
  })
})
