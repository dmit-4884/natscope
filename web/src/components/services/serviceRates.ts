import { useEffect, useState } from 'react'
import type { MicroDiscovery, MicroService } from '@/api/discovery'

interface Counters {
  requests: number
  errors: number
  processingNs: number
}

interface Sample {
  at: number
  counters: Map<string, { service: string; instance: string; endpoint: string; subject: string; started?: number } & Counters>
}

export interface EndpointWindow extends Counters {
  service: string
  instance: string
  endpoint: string
  subject: string
  seconds: number
}

export interface WindowTotal extends Counters {
  seconds: number
}

export type Health = 'ok' | 'degraded' | 'failing' | 'idle' | 'unknown'

const DEGRADED_ERROR_SHARE = 0.01
const FAILING_ERROR_SHARE = 0.1

export function sampleOf(services: MicroService[], at: number): Sample {
  const counters: Sample['counters'] = new Map()
  for (const service of services) {
    for (const instance of service.instances) {
      for (const e of instance.endpoints) {
        if (!e.stats) continue
        counters.set([service.name, instance.id, e.name, e.subject].join('\u0000'), {
          service: service.name,
          instance: instance.id,
          endpoint: e.name,
          subject: e.subject,
          started: instance.started,
          requests: e.stats.num_requests,
          errors: e.stats.num_errors,
          processingNs: e.stats.processing_time_ns,
        })
      }
    }
  }
  return { at, counters }
}

export function windowsBetween(before: Sample, after: Sample): EndpointWindow[] {
  const seconds = (after.at - before.at) / 1000
  if (seconds <= 0) return []
  const windows: EndpointWindow[] = []
  for (const [key, now] of after.counters) {
    const then = before.counters.get(key)
    const startedInWindow = now.started != null && now.started >= before.at
    if (!then && !startedInWindow) continue
    const restarted = !then || now.requests < then.requests
    windows.push({
      ...now,
      requests: restarted ? now.requests : now.requests - then.requests,
      errors: restarted ? now.errors : now.errors - then.errors,
      processingNs: restarted ? now.processingNs : now.processingNs - then.processingNs,
      seconds,
    })
  }
  return windows
}

export function sumWindows(windows: EndpointWindow[], match: (w: EndpointWindow) => boolean): WindowTotal | undefined {
  let total: WindowTotal | undefined
  for (const w of windows) {
    if (!match(w)) continue
    total ??= { requests: 0, errors: 0, processingNs: 0, seconds: w.seconds }
    total.requests += w.requests
    total.errors += w.errors
    total.processingNs += w.processingNs
  }
  return total
}

export function healthOf(window: WindowTotal | undefined): Health {
  if (!window) return 'unknown'
  if (window.requests === 0) return 'idle'
  const share = window.errors / window.requests
  if (share > FAILING_ERROR_SHARE) return 'failing'
  if (share > DEGRADED_ERROR_SHARE) return 'degraded'
  return 'ok'
}

const remembered = new Map<string, { sample: Sample; windows: EndpointWindow[] | null }>()

export function useServiceWindows(
  connectionId: string,
  data: MicroDiscovery | undefined,
  updatedAt: number,
): EndpointWindow[] | null {
  const [shown, setShown] = useState(() => ({ connectionId, windows: remembered.get(connectionId)?.windows ?? null }))

  useEffect(() => {
    if (!data || !updatedAt) return
    const last = remembered.get(connectionId)
    if (last?.sample.at === updatedAt) {
      setShown((prev) => (prev.connectionId === connectionId && prev.windows === last.windows ? prev : { connectionId, windows: last.windows }))
      return
    }
    const next = sampleOf(data.services, updatedAt)
    const windows = last && next.counters.size > 0 ? windowsBetween(last.sample, next) : null
    remembered.set(connectionId, { sample: next, windows })
    setShown({ connectionId, windows })
  }, [connectionId, data, updatedAt])

  return shown.connectionId === connectionId ? shown.windows : (remembered.get(connectionId)?.windows ?? null)
}
