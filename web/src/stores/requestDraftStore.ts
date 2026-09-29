import { useCallback } from 'react'
import { create } from 'zustand'
import { createJSONStorage, persist } from 'zustand/middleware'
import { z } from 'zod'
import { safeGetItem, safeRemoveItem, safeSetItem } from '@/utils/safeStorage'

const DEFAULT_REQUEST_TIMEOUT_MS = 5000

const MAX_RECENT_SUBJECTS = 10

const requestDraftSchema = z.object({
  subject: z.string(),
  payload: z.string(),
  headers: z.array(z.object({ key: z.string(), value: z.string() })),
  timeoutMs: z.number().int().positive(),
  recentSubjects: z.array(z.string()),
  replyTypes: z.record(z.string(), z.string()),
})

export type RequestDraft = z.infer<typeof requestDraftSchema>

const DEFAULT_DRAFT: RequestDraft = {
  subject: '',
  payload: '',
  headers: [],
  timeoutMs: DEFAULT_REQUEST_TIMEOUT_MS,
  recentSubjects: [],
  replyTypes: {},
}

interface RequestDraftState {
  drafts: Record<string, RequestDraft>
  update: (connectionId: string, patch: Partial<RequestDraft>) => void
  clearAll: () => void
}

const useRequestDraftStore = create<RequestDraftState>()(
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
      name: 'natscope:request-draft',
      storage: createJSONStorage(() => ({ getItem: safeGetItem, setItem: safeSetItem, removeItem: safeRemoveItem })),
      partialize: (state) => ({ drafts: state.drafts }),
      onRehydrateStorage: () => (state) => {
        if (!state) return
        const next: Record<string, RequestDraft> = {}
        if (typeof state.drafts === 'object' && state.drafts !== null) {
          for (const [connectionId, draft] of Object.entries(state.drafts)) {
            const parsed = requestDraftSchema.safeParse({ ...DEFAULT_DRAFT, ...draft })
            if (parsed.success) next[connectionId] = parsed.data
          }
        }
        state.drafts = next
      },
    },
  ),
)

export function useRequestDraft(connectionId: string) {
  const draft = useRequestDraftStore((s) => s.drafts[connectionId]) ?? DEFAULT_DRAFT
  const update = useRequestDraftStore((s) => s.update)
  const patch = useCallback((p: Partial<RequestDraft>) => update(connectionId, p), [update, connectionId])
  return [draft, patch] as const
}

export function withRecentSubject(recent: string[], subject: string): string[] {
  return [subject, ...recent.filter((s) => s !== subject)].slice(0, MAX_RECENT_SUBJECTS)
}

export function clearAllRequestDrafts(): void {
  useRequestDraftStore.getState().clearAll()
}
