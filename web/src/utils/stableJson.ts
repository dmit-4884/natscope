function sortKeys(value: unknown): unknown {
  if (Array.isArray(value)) return value.map(sortKeys)
  if (value === null || typeof value !== 'object') return value

  const source = value as Record<string, unknown>
  const sorted: Record<string, unknown> = {}
  for (const key of Object.keys(source).sort()) {
    if (source[key] === undefined) continue
    sorted[key] = sortKeys(source[key])
  }
  return sorted
}

export function stableJson(value: unknown, space?: number): string {
  return JSON.stringify(sortKeys(value), null, space)
}
