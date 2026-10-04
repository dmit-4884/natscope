import { useCallback } from 'react'
import { create } from 'zustand'
import { createJSONStorage, persist } from 'zustand/middleware'
import { z } from 'zod'
import { safeGetItem, safeRemoveItem, safeSetItem } from '@/utils/safeStorage'

const MAX_RECENT_SUBJECTS = 10

const subscribeDraftSchema = z.object({
  subjects: z.array(z.string()),
  recentSubjects: z.array(z.string()),
})

export type SubscribeDraft = z.infer<typeof subscribeDraftSchema>

const DEFAULT_DRAFT: SubscribeDraft = {
  subjects: [],
  recentSubjects: [],
}

interface SubscribeDraftState {
  drafts: Record<string, SubscribeDraft>
  update: (connectionId: string, patch: Partial<SubscribeDraft>) => void
  clearAll: () => void
}

const useSubscribeDraftStore = create<SubscribeDraftState>()(
  persist(
    (set) => ({
      drafts: {},
      update: (connectionId, patch) =>
        set((state) => ({
          drafts: {
            ...state.drafts,
            [connectionId]: { ...DEFAULT_DRAFT, ...state.drafts[connectionId], ...patch },
          },
        })),
      clearAll: () => set({ drafts: {} }),
    }),
    {
      name: 'natscope:subscribe-draft',
      storage: createJSONStorage(() => ({ getItem: safeGetItem, setItem: safeSetItem, removeItem: safeRemoveItem })),
      partialize: (state) => ({ drafts: state.drafts }),
      onRehydrateStorage: () => (state) => {
        if (!state) return
        const next: Record<string, SubscribeDraft> = {}
        if (typeof state.drafts === 'object' && state.drafts !== null) {
          for (const [connectionId, draft] of Object.entries(state.drafts)) {
            const parsed = subscribeDraftSchema.safeParse({ ...DEFAULT_DRAFT, ...draft })
            if (parsed.success) next[connectionId] = parsed.data
          }
        }
        state.drafts = next
      },
    },
  ),
)

export function useSubscribeDraft(connectionId: string) {
  const draft = useSubscribeDraftStore((s) => s.drafts[connectionId]) ?? DEFAULT_DRAFT
  const update = useSubscribeDraftStore((s) => s.update)
  const patch = useCallback((p: Partial<SubscribeDraft>) => update(connectionId, p), [update, connectionId])
  return [draft, patch] as const
}

export function withRecentSubjects(recent: string[], subjects: string[]): string[] {
  return [...new Set([...subjects, ...recent])].slice(0, MAX_RECENT_SUBJECTS)
}

export function clearAllSubscribeDrafts(): void {
  useSubscribeDraftStore.getState().clearAll()
}
