const UNIT_NS: Record<string, number> = { d: 86_400_000_000_000, h: 3_600_000_000_000, m: 60_000_000_000, s: 1_000_000_000, ms: 1_000_000 }
const DURATION = /^(?:\d+(?:\.\d+)?(?:ms|d|h|m|s))+$/
const PART = /(\d+(?:\.\d+)?)(ms|d|h|m|s)/g
const EXAMPLE_ERROR = 'Use a duration like 30s, 5m, 12h or 7d (0 for none)'

export interface ParsedDuration {
  ns?: number
  error?: string
}

export function parseDurationToNs(text: string): ParsedDuration {
  const value = text.trim()
  if (value === '' || value === '0') return { ns: 0 }
  if (!DURATION.test(value)) return { error: EXAMPLE_ERROR }
  let ns = 0
  for (const [, amount, unit] of value.matchAll(PART)) ns += Number(amount) * UNIT_NS[unit]
  return { ns: Math.round(ns) }
}
