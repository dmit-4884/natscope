import { create } from 'zustand'

interface SchemaBrowserState {
  search: string
  sourceFilter: string
  showImported: boolean
  selectedId: string | null
  setSearch: (search: string) => void
  setSourceFilter: (sourceFilter: string) => void
  setShowImported: (showImported: boolean) => void
  setSelectedId: (selectedId: string | null) => void
}

const INITIAL = { search: '', sourceFilter: '', showImported: false, selectedId: null }

export const useSchemaBrowserStore = create<SchemaBrowserState>()((set) => ({
  ...INITIAL,
  setSearch: (search) => set({ search }),
  setSourceFilter: (sourceFilter) => set({ sourceFilter }),
  setShowImported: (showImported) => set({ showImported }),
  setSelectedId: (selectedId) => set({ selectedId }),
}))

export function resetSchemaBrowser(): void {
  useSchemaBrowserStore.setState(INITIAL)
}
