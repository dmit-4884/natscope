/**
 * Jump-to-time helpers: convert between the picker's local `YYYY-MM-DDTHH:mm`
 * string, a Date, and epoch ms.
 */

/**
 * Parse a local `YYYY-MM-DDTHH:mm` to epoch ms (null if empty/unparseable); no
 * offset means browser-local time.
 */
export function parseStartDate(value: string | null | undefined): number | null {
  if (!value) return null
  const ms = new Date(value).getTime()
  return Number.isNaN(ms) ? null : ms
}

/** Format a Date as a local `YYYY-MM-DDTHH:mm` value for a datetime input. */
export function toDatetimeLocal(date: Date): string {
  const pad = (n: number) => String(n).padStart(2, '0')
  return (
    `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}` +
    `T${pad(date.getHours())}:${pad(date.getMinutes())}`
  )
}

/** A datetime-local string for `minutesAgo` minutes before now (local time). */
export function minutesAgoLocal(minutesAgo: number, now: Date = new Date()): string {
  return toDatetimeLocal(new Date(now.getTime() - minutesAgo * 60_000))
}
