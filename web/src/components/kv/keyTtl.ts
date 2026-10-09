const UNIT_NS: Record<string, number> = { h: 3_600_000_000_000, m: 60_000_000_000, s: 1_000_000_000, ms: 1_000_000 }
const DURATION = /^(?:\d+(?:\.\d+)?(?:ms|h|m|s))+$/
const PART = /(\d+(?:\.\d+)?)(ms|h|m|s)/g
const MIN_TTL_NS = 1_000_000_000

export interface KeyTtl {
  ns?: number
  error?: string
}

export function parseKeyTtl(text: string): KeyTtl {
  const value = text.trim()
  if (!value) return {}
  let ns: number
  if (/^\d+$/.test(value)) {
    ns = Number(value) * UNIT_NS.s
  } else if (DURATION.test(value)) {
    ns = 0
    for (const [, amount, unit] of value.matchAll(PART)) ns += Number(amount) * UNIT_NS[unit]
    ns = Math.round(ns)
  } else {
    return { error: 'TTL must be a duration like 30s, 5m or 1h' }
  }
  if (ns < MIN_TTL_NS) return { error: 'TTL must be at least 1s' }
  return { ns }
}
