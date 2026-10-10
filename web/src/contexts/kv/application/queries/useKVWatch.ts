import { useEffect, useState } from 'react'
import { useQueryClient, type QueryClient } from '@tanstack/react-query'
import * as api from '@/api/management'
import { getErrorMessage } from '@/api/errors'
import type { KVChange, KVKeyList } from '@/types/management'
import { kvKeys } from './kvKeys'

export type KVWatchStatus = 'off' | 'starting' | 'live' | 'stopped'

const MAX_CHANGES = 200
const KEY_LIST_LIMIT = 1000
const BUCKETS_REFRESH_MS = 5_000
const NO_CHANGES: KVChange[] = []

interface WatchState {
  key: string
  status: 'live' | 'stopped'
  error?: string
  changes: KVChange[]
}

function applyChanges(queryClient: QueryClient, connectionId: string, bucket: string, filter: string, changes: KVChange[]) {
  queryClient.setQueryData<KVKeyList>(kvKeys.keyList(connectionId, bucket, filter), (list) => {
    if (!list) return list
    const keys = new Set(list.keys)
    let truncated = list.truncated
    let changed = false
    for (const c of changes) {
      if (c.operation !== 'put') {
        changed = keys.delete(c.key) || changed
      } else if (keys.has(c.key)) {
        continue
      } else if (truncated || keys.size >= KEY_LIST_LIMIT) {
        changed = changed || !truncated
        truncated = true
      } else {
        keys.add(c.key)
        changed = true
      }
    }
    return changed ? { keys: [...keys], truncated } : list
  })
  for (const key of new Set(changes.map((c) => c.key))) {
    queryClient.invalidateQueries({ queryKey: kvKeys.key(connectionId, bucket, key) })
    queryClient.invalidateQueries({ queryKey: kvKeys.history(connectionId, bucket, key) })
  }
}

export function useKVWatch(connectionId: string | undefined, bucket: string | undefined, filter: string, enabled: boolean) {
  const queryClient = useQueryClient()
  const [attempt, setAttempt] = useState(0)
  const [state, setState] = useState<WatchState | null>(null)
  const watchKey = `${connectionId}|${bucket}|${filter}|${attempt}`

  useEffect(() => {
    if (!enabled || !connectionId || !bucket) return
    const key = `${connectionId}|${bucket}|${filter}|${attempt}`
    const controller = new AbortController()
    let bucketsRefreshedAt = 0
    const update = (next: (prev: KVChange[]) => Omit<WatchState, 'key'>) =>
      setState((prev) => ({ key, ...next(prev?.key === key ? prev.changes : NO_CHANGES) }))

    void (async () => {
      try {
        let started = false
        for await (const batch of api.watchKV(connectionId, bucket, filter, controller.signal)) {
          if (!started) {
            started = true
            queryClient.invalidateQueries({ queryKey: kvKeys.keys(connectionId, bucket) })
          }
          update((changes) => ({ status: 'live', changes: [...batch].reverse().concat(changes).slice(0, MAX_CHANGES) }))
          if (batch.length === 0) continue
          applyChanges(queryClient, connectionId, bucket, filter, batch)
          if (Date.now() - bucketsRefreshedAt > BUCKETS_REFRESH_MS) {
            bucketsRefreshedAt = Date.now()
            queryClient.invalidateQueries({ queryKey: kvKeys.buckets(connectionId) })
          }
        }
        if (!controller.signal.aborted) update((changes) => ({ status: 'stopped', error: 'The watch ended', changes }))
      } catch (err) {
        if (!controller.signal.aborted) update((changes) => ({ status: 'stopped', error: getErrorMessage(err), changes }))
      }
    })()
    return () => controller.abort()
  }, [enabled, connectionId, bucket, filter, attempt, queryClient])

  const current = enabled && state?.key === watchKey ? state : null
  const status: KVWatchStatus = !enabled ? 'off' : current?.status ?? 'starting'
  return {
    status,
    error: current?.error,
    changes: current?.changes ?? NO_CHANGES,
    restart: () => setAttempt((n) => n + 1),
  }
}
