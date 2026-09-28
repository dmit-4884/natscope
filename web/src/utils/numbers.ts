export function parseIntOr(value: string, fallback: number): number {
  const parsed = parseInt(value, 10)
  return Number.isNaN(parsed) ? fallback : parsed
}

export const INT32_MIN = -2147483648
export const INT32_MAX = 2147483647

export function clampInt32(value: number): number {
  return Math.min(INT32_MAX, Math.max(INT32_MIN, value))
}
