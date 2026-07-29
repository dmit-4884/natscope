import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { z } from 'zod'

const bookmarkSchema = z.object({
  id: z.string(),
  connectionId: z.string(),
  streamName: z.string(),
  sequence: z.number(),
  subject: z.string(),
  note: z.string().optional(),
  createdAt: z.string(),
  dataPreview: z.string().optional(),
})

const bookmarksArraySchema = z.array(bookmarkSchema)

export type Bookmark = z.infer<typeof bookmarkSchema>

/** Safety cap; oldest evicted past this to bound localStorage growth. */
const MAX_BOOKMARKS = 500

interface BookmarkState {
  bookmarks: Bookmark[]
  addBookmark: (bookmark: Omit<Bookmark, 'id' | 'createdAt'>) => void
  removeBookmark: (id: string) => void
  updateNote: (id: string, note: string) => void
  isBookmarked: (connectionId: string, streamName: string, sequence: number) => boolean
  getBookmark: (connectionId: string, streamName: string, sequence: number) => Bookmark | undefined
  getConnectionBookmarks: (connectionId: string) => Bookmark[]
  clearConnectionBookmarks: (connectionId: string) => void
}

export const useBookmarkStore = create<BookmarkState>()(
  persist(
    (set, get) => ({
      bookmarks: [],

      addBookmark: (bookmarkData) => {
        const { bookmarks } = get()
        const newBookmark: Bookmark = {
          ...bookmarkData,
          id: crypto.randomUUID(),
          createdAt: new Date().toISOString(),
        }
        const next = [...bookmarks, newBookmark]
        // FIFO eviction at cap — keeps newest MAX_BOOKMARKS entries.
        const trimmed = next.length > MAX_BOOKMARKS ? next.slice(next.length - MAX_BOOKMARKS) : next
        set({ bookmarks: trimmed })
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

      getConnectionBookmarks: (connectionId) => {
        const { bookmarks } = get()
        return bookmarks.filter((b) => b.connectionId === connectionId)
      },

      clearConnectionBookmarks: (connectionId) => {
        const { bookmarks } = get()
        set({ bookmarks: bookmarks.filter((b) => b.connectionId !== connectionId) })
      },
    }),
    {
      name: 'natscope-bookmarks',
      onRehydrateStorage: () => (state) => {
        if (!state) return
        const parsed = bookmarksArraySchema.safeParse(state.bookmarks)
        if (!parsed.success) {
           
          console.warn('[bookmarkStore] schema mismatch, resetting', parsed.error.message)
          state.bookmarks = []
        }
      },
    },
  ),
)
