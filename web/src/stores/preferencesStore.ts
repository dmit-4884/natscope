import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { z } from 'zod'

const persistedShape = z.object({
  panelSizes: z.record(z.string(), z.number()),
  collapsedPanels: z.record(z.string(), z.boolean()),
  recentCommands: z.array(z.string()),
})

interface PreferencesState {
  // Panel sizes (persisted)
  panelSizes: Record<string, number>
  setPanelSize: (panelId: string, size: number) => void
  getPanelSize: (panelId: string, defaultSize: number) => number

  // Collapsed panels
  collapsedPanels: Record<string, boolean>
  togglePanelCollapsed: (panelId: string) => void
  isPanelCollapsed: (panelId: string) => boolean

  // Recent commands for command palette
  recentCommands: string[]
  addRecentCommand: (commandId: string) => void
  getRecentCommands: () => string[]
}

const MAX_RECENT_COMMANDS = 5

export const usePreferencesStore = create<PreferencesState>()(
  persist(
    (set, get) => ({
      // Panel sizes
      panelSizes: {},
      setPanelSize: (panelId, size) =>
        set((state) => ({
          panelSizes: { ...state.panelSizes, [panelId]: size },
        })),
      getPanelSize: (panelId, defaultSize) => {
        const { panelSizes } = get()
        return panelSizes[panelId] ?? defaultSize
      },

      // Collapsed panels
      collapsedPanels: {},
      togglePanelCollapsed: (panelId) =>
        set((state) => ({
          collapsedPanels: {
            ...state.collapsedPanels,
            [panelId]: !state.collapsedPanels[panelId],
          },
        })),
      isPanelCollapsed: (panelId) => {
        const { collapsedPanels } = get()
        return collapsedPanels[panelId] ?? false
      },

      // Recent commands
      recentCommands: [],
      addRecentCommand: (commandId) =>
        set((state) => {
          const filtered = state.recentCommands.filter((id) => id !== commandId)
          return {
            recentCommands: [commandId, ...filtered].slice(0, MAX_RECENT_COMMANDS),
          }
        }),
      getRecentCommands: () => get().recentCommands,
    }),
    {
      name: 'natscope-preferences',
      onRehydrateStorage: () => (state) => {
        if (!state) return
        const snapshot = {
          panelSizes: state.panelSizes,
          collapsedPanels: state.collapsedPanels,
          recentCommands: state.recentCommands,
        }
        const parsed = persistedShape.safeParse(snapshot)
        if (!parsed.success) {

          console.warn('[preferencesStore] schema mismatch, resetting', parsed.error.message)
          state.panelSizes = {}
          state.collapsedPanels = {}
          state.recentCommands = []
        }
      },
    },
  ),
)
