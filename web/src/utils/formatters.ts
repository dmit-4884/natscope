// Unified formatting utilities.

import { format } from 'date-fns'

/**
 * The UI is English-only, so every date and number is pinned to this locale
 * rather than the browser's — otherwise an ru-RU browser mixes `30.07.2026`
 * into English labels.
 */
const UI_LOCALE = 'en-US'

/** Bytes -> human-readable (e.g. "1.5 MB"); raw byte counts stay integers. */
export function formatBytes(bytes: number, precision = 2): string {
  if (bytes === 0) return '0 B'
  if (bytes < 0 || !Number.isFinite(bytes)) return 'Unlimited'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.min(Math.floor(Math.log(bytes) / Math.log(k)), sizes.length - 1)
  return `${(bytes / Math.pow(k, i)).toFixed(i === 0 ? 0 : precision)} ${sizes[i]}`
}

/** Number -> grouped decimal (e.g. "1,234"), pinned to the UI locale. */
export function formatCount(value: number): string {
  return value.toLocaleString(UI_LOCALE)
}

/** Bytes/sec -> human-readable (e.g. "1.5 MB/s"). */
export function formatBytesPerSecond(bytes: number): string {
  return `${formatBytes(bytes)}/s`
}

/** Large number -> K/M/B suffix (e.g. "1.2M"). */
export function formatNumber(num: number): string {
  if (num < 1000) return num.toString()
  if (num < 1000000) return (num / 1000).toFixed(1) + 'K'
  if (num < 1000000000) return (num / 1000000).toFixed(1) + 'M'
  return (num / 1000000000).toFixed(1) + 'B'
}

/** Date input type - supports Date, ISO string, or Unix milliseconds */
export type DateInput = Date | string | number

/** Date | ISO string | Unix ms -> Date. */
export function toDate(date: DateInput): Date {
  if (date instanceof Date) return date
  if (typeof date === 'number') return new Date(date)
  return new Date(date)
}

/** Format a timestamp per user's configurable format setting. */
export function formatTimestamp(
  timestamp: DateInput,
  fmt: 'relative' | 'absolute' | 'iso' = 'relative',
): string {
  const d = toDate(timestamp)
  switch (fmt) {
    case 'relative': {
      const now = Date.now()
      const diff = now - d.getTime()
      if (diff < 0) return 'just now'
      if (diff < 60000) return `${Math.floor(diff / 1000)}s ago`
      if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`
      if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`
      return `${Math.floor(diff / 86400000)}d ago`
    }
    case 'iso':
      return format(d, 'yyyy-MM-dd HH:mm:ss')
    case 'absolute':
    default:
      return format(d, 'dd.MM.yy HH:mm:ss')
  }
}

/** Timestamp -> time with ms (HH:mm:ss.SSS); for live messages. */
export function formatTimeWithMs(timestamp: DateInput): string {
  return format(toDate(timestamp), 'HH:mm:ss.SSS')
}

/** Timestamp -> 24-hour time (HH:mm:ss). */
export function formatTime(timestamp: DateInput): string {
  return format(toDate(timestamp), 'HH:mm:ss')
}

/** Date -> ISO date (yyyy-MM-dd). */
export function formatDate(date: DateInput): string {
  return format(toDate(date), 'yyyy-MM-dd')
}

/** Date -> compact month/day (MM-dd); for dense history columns. */
export function formatMonthDay(date: DateInput): string {
  return format(toDate(date), 'MM-dd')
}

/** Date -> full datetime (yyyy-MM-dd HH:mm:ss); for debug. */
export function formatDateTime(date: DateInput): string {
  return format(toDate(date), 'yyyy-MM-dd HH:mm:ss')
}

/** Seconds -> human-readable duration (e.g. "1h 30m"). */
export function formatDuration(seconds: number): string {
  if (seconds < 60) return `${seconds}s`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ${seconds % 60}s`
  const h = Math.floor(seconds / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  return `${h}h ${m}m`
}

/** Nanoseconds -> human-readable (e.g. "1.2ms"); for latency. */
export function formatNanoseconds(ns: number): string {
  if (ns < 1000) return `${ns}ns`
  if (ns < 1000000) return `${(ns / 1000).toFixed(1)}µs`
  if (ns < 1000000000) return `${(ns / 1000000).toFixed(1)}ms`
  return `${(ns / 1000000000).toFixed(2)}s`
}

/**
 * Nanosecond NATS duration (max_age, ack_wait) -> "1d 2h 30m".
 * Returns "Unlimited" for <= 0.
 */
export function formatNsDuration(ns: number): string {
  if (ns <= 0) return 'Unlimited'

  const seconds = Math.floor(ns / 1_000_000_000)
  if (seconds < 60) return `${seconds}s`

  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) {
    const remainingSec = seconds % 60
    return remainingSec > 0 ? `${minutes}m ${remainingSec}s` : `${minutes}m`
  }

  const hours = Math.floor(minutes / 60)
  if (hours < 24) {
    const remainingMin = minutes % 60
    return remainingMin > 0 ? `${hours}h ${remainingMin}m` : `${hours}h`
  }

  const days = Math.floor(hours / 24)
  const remainingHours = hours % 24
  return remainingHours > 0 ? `${days}d ${remainingHours}h` : `${days}d`
}
