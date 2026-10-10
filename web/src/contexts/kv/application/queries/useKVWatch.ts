import { useEffect, useRef, useState } from 'react'
import { Code } from '@connectrpc/connect'
import { useQueryClient, type QueryClient } from '@tanstack/react-query'
import * as api from '@/api/management'
import { getErrorMessage, isErrorCode } from '@/api/errors'
import type { KVChange, KVKeyList } from '@/types/management'
import { kvKeys } from './kvKeys'

export type KVWatchStatus = 'off' | 'starting' | 'live' | 'reconnecting' | 'stopped'

const MAX_CHANGES = 200
const KEY_LIST_LIMIT = 1000
const BUCKETS_REFRESH_MS = 5_000
const RETRY_FIRST_MS = 1_000
const RETRY_MAX_MS = 15_000
const FINAL_CODES = [Code.NotFound, Code.PermissionDenied, Code.InvalidArgument, Code.FailedPrecondition, Code.Unauthenticated]
const NO_CHANGES: KVChange[] = []

interface WatchState {
  key: string
  status: 'live' | 'reconnecting' | 'stopped'
  error?: string
  changes: KVChange[]
}

function isFinal(err: unknown): boolean {
  return FINAL_CODES.some((code) => isErrorCode(err, code))
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
  const failuresRef = useRef(0)
  const watchKey = `${connectionId}|${bucket}|${filter}|${attempt}`

  useEffect(() => {
    if (!enabled || !connectionId || !bucket) return
    const key = `${connectionId}|${bucket}|${filter}|${attempt}`
    const controller = new AbortController()
    let retryTimer: ReturnType<typeof setTimeout> | undefined
    let bucketsRefreshedAt = 0
    const update = (next: (prev: KVChange[]) => Omit<WatchState, 'key'>) =>
      setState((prev) => ({ key, ...next(prev?.key === key ? prev.changes : NO_CHANGES) }))
    const reconnect = (error: string) => {
      failuresRef.current += 1
      const delay = Math.min(RETRY_MAX_MS, RETRY_FIRST_MS * 2 ** (failuresRef.current - 1))
      update((changes) => ({ status: 'reconnecting', error, changes }))
      retryTimer = setTimeout(() => setAttempt((n) => n + 1), delay)
    }

    void (async () => {
      try {
        let started = false
        for await (const batch of api.watchKV(connectionId, bucket, filter, controller.signal)) {
          if (!started) {
            started = true
            failuresRef.current = 0
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
        if (!controller.signal.aborted) reconnect('The watch ended')
      } catch (err) {
        if (controller.signal.aborted) return
        if (isFinal(err)) update((changes) => ({ status: 'stopped', error: getErrorMessage(err), changes }))
        else reconnect(getErrorMessage(err))
      }
    })()
    return () => {
      controller.abort()
      clearTimeout(retryTimer)
    }
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
