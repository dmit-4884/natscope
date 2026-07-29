export function parseIntOr(value: string, fallback: number): number {
  const parsed = parseInt(value, 10)
  return Number.isNaN(parsed) ? fallback : parsed
}
