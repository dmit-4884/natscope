import { create } from 'zustand'

/** Legacy stats shape (backwards compat). */
interface LegacyStats {
  messagesReceived: number
  messagesDropped: number
  msgPerSecond: number
  isConnected: boolean
}

interface LiveStatsStore {
  stats: LegacyStats | null
  setStats: (stats: LegacyStats | null) => void
}

/** Live statistics store; keeps the legacy interface. */
export const useLiveStatsStore = create<LiveStatsStore>((set) => ({
  stats: null,

  setStats: (stats) => set({ stats }),
}))
