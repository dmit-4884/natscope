import { useMemo } from 'react'
import { create, type StoreApi, type UseBoundStore } from 'zustand'
import { persist, createJSONStorage, type StateStorage } from 'zustand/middleware'
import type { ZodType } from 'zod'
import { streamKey, type StreamScope } from './scope'

/** Entry wrapper: payload + last-touched timestamp for TTL pruning. */
interface ScopedEntry<T> {
  data: T
  updatedAt: number
}

interface ScopedState<T> {
  entries: Record<string, ScopedEntry<T>>
}

interface StoreActions<T> {
  /** Internal — consumers go through the {@link StreamScopedStore} façade. */
  __setEntry: (key: string, partial: Partial<T>, defaults: T) => void
  __clearAll: () => void
}

export interface StreamScopedStore<T extends object> {
  /**
   * Hook -> [entry (defaults if absent), partial patcher]. Named useEntry
   * (not `use`) so the hooks-rules lint recognizes it as a hook.
   */
  useEntry(scope: StreamScope): readonly [T, (patch: Partial<T>) => void]

  /** Imperative read — useful in callbacks/effects without subscribing. */
  get(scope: StreamScope): T

  /** Imperative shallow merge. */
  set(scope: StreamScope, patch: Partial<T>): void

  /** Drop every entry (e.g. on logout). */
  clearAll(): void

  /** Underlying Zustand store — exposed for advanced use only. */
  _internal: UseBoundStore<StoreApi<ScopedState<T> & StoreActions<T>>>
}

export interface CreateStreamScopedStoreOptions<T> {
  /** Persistence key (final localStorage key is `natscope:stream-scope:${name}`). */
  name: string
  /** Default value returned for scopes with no entry yet. */
  defaults: T
  /** Schema version — bump when shape changes incompatibly. Defaults to 1. */
  version?: number
  /** TTL after last update; older entries pruned on rehydrate. Default 30 days. */
  ttlMs?: number
  /** Override storage (mainly for tests). */
  storage?: StateStorage
  schema?: ZodType<T>
}

const DEFAULT_TTL_MS = 30 * 24 * 60 * 60 * 1000

function isScopedEntry(value: unknown): value is ScopedEntry<unknown> {
  if (typeof value !== 'object' || value === null) return false
  const entry = value as Partial<ScopedEntry<unknown>>
  return (
    typeof entry.updatedAt === 'number' &&
    Number.isFinite(entry.updatedAt) &&
    typeof entry.data === 'object' &&
    entry.data !== null
  )
}

export function createStreamScopedStore<T extends object>(
  options: CreateStreamScopedStoreOptions<T>,
): StreamScopedStore<T> {
  const {
    name,
    defaults,
    version = 1,
    ttlMs = DEFAULT_TTL_MS,
    storage,
    schema,
  } = options

  const persistKey = `natscope:stream-scope:${name}`

  const useStore = create<ScopedState<T> & StoreActions<T>>()(
    persist(
      (set, get) => ({
        entries: {},

        __setEntry: (key, partial, defs) => {
          const prev = get().entries[key]?.data ?? defs
          set((state) => ({
            entries: {
              ...state.entries,
              [key]: { data: { ...prev, ...partial }, updatedAt: Date.now() },
            },
          }))
        },

        __clearAll: () => set({ entries: {} }),
      }),
      {
        name: persistKey,
        version,
        storage: createJSONStorage(() => storage ?? localStorage),
        onRehydrateStorage: () => (state) => {
          if (!state) return
          if (typeof state.entries !== 'object' || state.entries === null) {
            state.entries = {}
            return
          }
          const now = Date.now()
          const next: Record<string, ScopedEntry<T>> = {}
          for (const [k, entry] of Object.entries(state.entries)) {
            if (!isScopedEntry(entry) || now - entry.updatedAt >= ttlMs) continue
            if (schema) {
              const parsed = schema.safeParse({ ...defaults, ...(entry.data as object) })
              if (!parsed.success) continue
            }
            next[k] = entry as ScopedEntry<T>
          }
          state.entries = next
        },
      },
    ),
  )

  return {
    _internal: useStore,

    useEntry(scope) {
      const k = streamKey(scope)
      const stored = useStore((s) => s.entries[k]?.data)
      const setEntry = useStore((s) => s.__setEntry)
      const data = useMemo(() => (stored ? { ...defaults, ...stored } : defaults), [stored])
      const update = (patch: Partial<T>) => setEntry(k, patch, defaults)
      return [data, update] as const
    },

    get(scope) {
      const k = streamKey(scope)
      const stored = useStore.getState().entries[k]?.data
      return stored ? { ...defaults, ...stored } : defaults
    },

    set(scope, patch) {
      useStore.getState().__setEntry(streamKey(scope), patch, defaults)
    },

    clearAll() {
      useStore.getState().__clearAll()
    },
  }
}
