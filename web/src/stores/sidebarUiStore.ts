import { create } from 'zustand'

interface SidebarUiState {
  open: Record<string, boolean>
  filters: Record<string, string>
  setOpen: (key: string, open: boolean) => void
  setFilter: (key: string, filter: string) => void
}

export const useSidebarUiStore = create<SidebarUiState>()((set) => ({
  open: {},
  filters: {},
  setOpen: (key, open) => set((state) => ({ open: { ...state.open, [key]: open } })),
  setFilter: (key, filter) => set((state) => ({ filters: { ...state.filters, [key]: filter } })),
}))

export function resetSidebarUi(): void {
  useSidebarUiStore.setState({ open: {}, filters: {} })
}
