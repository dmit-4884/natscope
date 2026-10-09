import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { ReactNode } from 'react'
import { act, renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { KVChange, KVKeyList } from '@/types/management'
import { kvKeys } from './kvKeys'
import { useKVWatch } from './useKVWatch'

const watch = vi.hoisted(() => ({
  push: null as null | ((batch: KVChange[]) => void),
  fail: null as null | ((err: Error) => void),
  end: null as null | (() => void),
  calls: [] as Array<{ bucket: string; filter: string }>,
}))

vi.mock('@/api/management', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/api/management')>()),
  watchKV: (_conn: string, bucket: string, filter: string, signal: AbortSignal) => {
    watch.calls.push({ bucket, filter })
    const queue: KVChange[][] = []
    let wake: (() => void) | null = null
    let error: Error | null = null
    let done = false
    const notify = () => {
      wake?.()
      wake = null
    }
    watch.push = (batch) => {
      queue.push(batch)
      notify()
    }
    watch.fail = (err) => {
      error = err
      notify()
    }
    watch.end = () => {
      done = true
      notify()
    }
    signal.addEventListener('abort', () => {
      done = true
      notify()
    })
    return (async function* () {
      for (;;) {
        if (queue.length) {
          yield queue.shift()!
          continue
        }
        if (error) throw error
        if (done) return
        await new Promise<void>((resolve) => {
          wake = resolve
        })
      }
    })()
  },
}))

function change(key: string, operation: KVChange['operation'], revision: number): KVChange {
  return { key, operation, revision, created: 0, value: btoa('v'), size: 1 }
}

function setup(enabled = true, filter = '') {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  client.setQueryData<KVKeyList>(kvKeys.keyList('conn-1', 'CONFIG', filter), { keys: ['a', 'b'], truncated: false })
  const wrapper = ({ children }: { children: ReactNode }) => <QueryClientProvider client={client}>{children}</QueryClientProvider>
  const hook = renderHook(({ on }) => useKVWatch('conn-1', 'CONFIG', filter, on), { wrapper, initialProps: { on: enabled } })
  return { client, hook }
}

describe('useKVWatch', () => {
  beforeEach(() => {
    watch.push = null
    watch.fail = null
    watch.end = null
    watch.calls = []
  })

  it('stays off until it is turned on', () => {
    const { hook } = setup(false)

    expect(hook.result.current.status).toBe('off')
    expect(watch.calls).toHaveLength(0)
  })

  it('is live once the empty first batch arrives', async () => {
    const { hook } = setup(true, 'orders.*')
    expect(hook.result.current.status).toBe('starting')
    expect(watch.calls).toEqual([{ bucket: 'CONFIG', filter: 'orders.*' }])

    act(() => watch.push!([]))

    await waitFor(() => expect(hook.result.current.status).toBe('live'))
  })

  it('lists the newest change first and patches the key list', async () => {
    const { client, hook } = setup()
    act(() => watch.push!([]))
    act(() => watch.push!([change('c', 'put', 3), change('a', 'delete', 4)]))

    await waitFor(() => expect(hook.result.current.changes.map((c) => c.key)).toEqual(['a', 'c']))
    expect(client.getQueryData<KVKeyList>(kvKeys.keyList('conn-1', 'CONFIG', ''))?.keys).toEqual(['b', 'c'])
  })

  it('reports a watch that stops and starts again on restart', async () => {
    const { hook } = setup()
    act(() => watch.push!([]))
    act(() => watch.fail!(new Error('connection closed')))

    await waitFor(() => expect(hook.result.current.status).toBe('stopped'))
    expect(hook.result.current.error).toMatch(/connection closed/)

    act(() => hook.result.current.restart())

    expect(hook.result.current.status).toBe('starting')
    expect(watch.calls).toHaveLength(2)
  })

  it('stops the watch when it is turned off', async () => {
    const { hook } = setup()
    act(() => watch.push!([]))
    await waitFor(() => expect(hook.result.current.status).toBe('live'))

    hook.rerender({ on: false })

    expect(hook.result.current.status).toBe('off')
  })
})
