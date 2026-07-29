import { describe, it, expect, beforeEach } from 'vitest'
import { create } from 'zustand'

const MAX_RECENT_COMMANDS = 5

// Non-persisted version for testing
const useTestStore = create<{
  panelSizes: Record<string, number>
  setPanelSize: (panelId: string, size: number) => void
  getPanelSize: (panelId: string, defaultSize: number) => number
  collapsedPanels: Record<string, boolean>
  togglePanelCollapsed: (panelId: string) => void
  isPanelCollapsed: (panelId: string) => boolean
  recentCommands: string[]
  addRecentCommand: (commandId: string) => void
  getRecentCommands: () => string[]
}>((set, get) => ({
  panelSizes: {},
  setPanelSize: (panelId, size) =>
    set((state) => ({
      panelSizes: { ...state.panelSizes, [panelId]: size },
    })),
  getPanelSize: (panelId, defaultSize) => {
    const { panelSizes } = get()
    return panelSizes[panelId] ?? defaultSize
  },

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

  recentCommands: [],
  addRecentCommand: (commandId) =>
    set((state) => {
      const filtered = state.recentCommands.filter((id) => id !== commandId)
      return {
        recentCommands: [commandId, ...filtered].slice(0, MAX_RECENT_COMMANDS),
      }
    }),
  getRecentCommands: () => get().recentCommands,
}))

describe('preferencesStore', () => {
  beforeEach(() => {
    useTestStore.setState({
      panelSizes: {},
      collapsedPanels: {},
      recentCommands: [],
    })
  })

  describe('panelSizes', () => {
    it('sets and gets panel size', () => {
      useTestStore.getState().setPanelSize('sidebar', 300)
      expect(useTestStore.getState().getPanelSize('sidebar', 250)).toBe(300)
    })

    it('returns default for unset panel', () => {
      expect(useTestStore.getState().getPanelSize('unknown', 200)).toBe(200)
    })
  })

  describe('collapsedPanels', () => {
    it('defaults to not collapsed', () => {
      expect(useTestStore.getState().isPanelCollapsed('sidebar')).toBe(false)
    })

    it('toggles collapsed state', () => {
      useTestStore.getState().togglePanelCollapsed('sidebar')
      expect(useTestStore.getState().isPanelCollapsed('sidebar')).toBe(true)
      useTestStore.getState().togglePanelCollapsed('sidebar')
      expect(useTestStore.getState().isPanelCollapsed('sidebar')).toBe(false)
    })
  })

  describe('recentCommands', () => {
    it('adds recent command', () => {
      useTestStore.getState().addRecentCommand('cmd1')
      expect(useTestStore.getState().getRecentCommands()).toEqual(['cmd1'])
    })

    it('moves duplicate to front', () => {
      useTestStore.getState().addRecentCommand('cmd1')
      useTestStore.getState().addRecentCommand('cmd2')
      useTestStore.getState().addRecentCommand('cmd1')
      expect(useTestStore.getState().getRecentCommands()).toEqual(['cmd1', 'cmd2'])
    })

    it('limits to MAX_RECENT_COMMANDS', () => {
      for (let i = 0; i < 10; i++) {
        useTestStore.getState().addRecentCommand(`cmd${i}`)
      }
      expect(useTestStore.getState().getRecentCommands()).toHaveLength(MAX_RECENT_COMMANDS)
    })

    it('puts newest command first', () => {
      useTestStore.getState().addRecentCommand('old')
      useTestStore.getState().addRecentCommand('new')
      expect(useTestStore.getState().getRecentCommands()[0]).toBe('new')
    })
  })
})
