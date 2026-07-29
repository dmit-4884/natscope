import { create } from 'zustand'

interface ManagementState {
  // Active tab
  activeTab: 'streams' | 'consumers' | 'kv' | 'objects'
  setActiveTab: (tab: 'streams' | 'consumers' | 'kv' | 'objects') => void

  // Streams tab
  streamsExpandedGroups: Record<string, boolean>
  toggleStreamsGroup: (group: string) => void
  streamsIsCreating: boolean
  setStreamsIsCreating: (creating: boolean) => void
  streamsSelectedName: string | null
  setStreamsSelectedName: (name: string | null) => void

  // Consumers tab
  consumersSelectedStream: string | null
  setConsumersSelectedStream: (stream: string | null) => void
  consumersIsCreating: boolean
  setConsumersIsCreating: (creating: boolean) => void
  consumersSelectedName: string | null
  setConsumersSelectedName: (name: string | null) => void

  // KV tab
  kvIsCreating: boolean
  setKvIsCreating: (creating: boolean) => void
  kvSelectedBucket: string | null
  setKvSelectedBucket: (bucket: string | null) => void

  // Objects tab
  objectsIsCreating: boolean
  setObjectsIsCreating: (creating: boolean) => void
  objectsSelectedBucket: string | null
  setObjectsSelectedBucket: (bucket: string | null) => void

  // Reset all state
  reset: () => void
}

export const useManagementStore = create<ManagementState>((set) => ({
  // Active tab
  activeTab: 'streams',
  setActiveTab: (tab) => set({ activeTab: tab }),

  // Streams tab
  streamsExpandedGroups: { regular: true, kv: false },
  toggleStreamsGroup: (group) => set((state) => ({
    streamsExpandedGroups: {
      ...state.streamsExpandedGroups,
      [group]: !state.streamsExpandedGroups[group],
    },
  })),
  streamsIsCreating: false,
  setStreamsIsCreating: (creating) => set({ streamsIsCreating: creating }),
  streamsSelectedName: null,
  setStreamsSelectedName: (name) => set({ streamsSelectedName: name }),

  // Consumers tab
  consumersSelectedStream: null,
  setConsumersSelectedStream: (stream) => set({ consumersSelectedStream: stream }),
  consumersIsCreating: false,
  setConsumersIsCreating: (creating) => set({ consumersIsCreating: creating }),
  consumersSelectedName: null,
  setConsumersSelectedName: (name) => set({ consumersSelectedName: name }),

  // KV tab
  kvIsCreating: false,
  setKvIsCreating: (creating) => set({ kvIsCreating: creating }),
  kvSelectedBucket: null,
  setKvSelectedBucket: (bucket) => set({ kvSelectedBucket: bucket }),

  // Objects tab
  objectsIsCreating: false,
  setObjectsIsCreating: (creating) => set({ objectsIsCreating: creating }),
  objectsSelectedBucket: null,
  setObjectsSelectedBucket: (bucket) => set({ objectsSelectedBucket: bucket }),

  // Reset all state
  reset: () => set({
    activeTab: 'streams',
    streamsExpandedGroups: { regular: true, kv: false },
    streamsIsCreating: false,
    streamsSelectedName: null,
    consumersSelectedStream: null,
    consumersIsCreating: false,
    consumersSelectedName: null,
    kvIsCreating: false,
    kvSelectedBucket: null,
    objectsIsCreating: false,
    objectsSelectedBucket: null,
  }),
}))
