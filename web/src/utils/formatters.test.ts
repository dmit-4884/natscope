import { describe, it, expect } from 'vitest'
import {
  formatBytes,
  formatBytesPerSecond,
  formatNumber,
  toDate,
  formatTimestamp,
  formatDate,
  formatDateTime,
  formatDuration,
  formatNanoseconds,
  formatNsDuration,
} from './formatters'

describe('formatters', () => {
  describe('formatBytes', () => {
    it('formats zero bytes', () => {
      expect(formatBytes(0)).toBe('0 B')
    })

    it('formats bytes', () => {
      expect(formatBytes(500)).toBe('500.00 B')
    })

    it('formats kilobytes', () => {
      expect(formatBytes(1024)).toBe('1.00 KB')
      expect(formatBytes(1536)).toBe('1.50 KB')
    })

    it('formats megabytes', () => {
      expect(formatBytes(1048576)).toBe('1.00 MB')
    })

    it('formats gigabytes', () => {
      expect(formatBytes(1073741824)).toBe('1.00 GB')
    })

    it('formats terabytes', () => {
      expect(formatBytes(1099511627776)).toBe('1.00 TB')
    })

    it('respects precision', () => {
      expect(formatBytes(1536, 1)).toBe('1.5 KB')
      expect(formatBytes(1536, 0)).toBe('2 KB')
    })
  })

  describe('formatBytesPerSecond', () => {
    it('appends /s suffix', () => {
      expect(formatBytesPerSecond(1024)).toBe('1.00 KB/s')
    })
  })

  describe('formatNumber', () => {
    it('formats small numbers', () => {
      expect(formatNumber(42)).toBe('42')
      expect(formatNumber(999)).toBe('999')
    })

    it('formats thousands', () => {
      expect(formatNumber(1500)).toBe('1.5K')
    })

    it('formats millions', () => {
      expect(formatNumber(2500000)).toBe('2.5M')
    })

    it('formats billions', () => {
      expect(formatNumber(1500000000)).toBe('1.5B')
    })
  })

  describe('toDate', () => {
    it('returns Date unchanged', () => {
      const d = new Date(2024, 0, 1)
      expect(toDate(d)).toBe(d)
    })

    it('converts unix ms to Date', () => {
      const d = toDate(1700000000000)
      expect(d).toBeInstanceOf(Date)
      expect(d.getTime()).toBe(1700000000000)
    })

    it('converts ISO string to Date', () => {
      const d = toDate('2024-01-01T00:00:00Z')
      expect(d).toBeInstanceOf(Date)
    })
  })

  describe('formatTimestamp', () => {
    it('formats relative time', () => {
      const recent = Date.now() - 30000 // 30 seconds ago
      expect(formatTimestamp(recent, 'relative')).toBe('30s ago')
    })

    it('formats minutes ago', () => {
      const recent = Date.now() - 120000 // 2 minutes ago
      expect(formatTimestamp(recent, 'relative')).toBe('2m ago')
    })

    it('formats hours ago', () => {
      const recent = Date.now() - 7200000 // 2 hours ago
      expect(formatTimestamp(recent, 'relative')).toBe('2h ago')
    })

    it('formats days ago', () => {
      const recent = Date.now() - 172800000 // 2 days ago
      expect(formatTimestamp(recent, 'relative')).toBe('2d ago')
    })

    it('formats ISO format', () => {
      const result = formatTimestamp(new Date(2024, 0, 15, 10, 30, 45), 'iso')
      expect(result).toBe('2024-01-15 10:30:45')
    })

    it('formats absolute format', () => {
      const result = formatTimestamp(new Date(2024, 0, 15, 10, 30, 45), 'absolute')
      expect(result).toBe('15.01.24 10:30:45')
    })
  })

  describe('formatDate', () => {
    it('formats as yyyy-MM-dd', () => {
      expect(formatDate(new Date(2024, 0, 15))).toBe('2024-01-15')
    })
  })

  describe('formatDateTime', () => {
    it('formats full datetime', () => {
      expect(formatDateTime(new Date(2024, 0, 15, 10, 30, 45))).toBe('2024-01-15 10:30:45')
    })
  })

  describe('formatDuration', () => {
    it('formats seconds only', () => {
      expect(formatDuration(45)).toBe('45s')
    })

    it('formats minutes and seconds', () => {
      expect(formatDuration(125)).toBe('2m 5s')
    })

    it('formats hours and minutes', () => {
      expect(formatDuration(3720)).toBe('1h 2m')
    })
  })

  describe('formatNanoseconds', () => {
    it('formats nanoseconds', () => {
      expect(formatNanoseconds(500)).toBe('500ns')
    })

    it('formats microseconds', () => {
      expect(formatNanoseconds(5000)).toBe('5.0µs')
    })

    it('formats milliseconds', () => {
      expect(formatNanoseconds(5000000)).toBe('5.0ms')
    })

    it('formats seconds', () => {
      expect(formatNanoseconds(1500000000)).toBe('1.50s')
    })
  })

  describe('formatNsDuration', () => {
    it('returns Unlimited for zero', () => {
      expect(formatNsDuration(0)).toBe('Unlimited')
    })

    it('returns Unlimited for negative', () => {
      expect(formatNsDuration(-1)).toBe('Unlimited')
    })

    it('formats seconds', () => {
      expect(formatNsDuration(30_000_000_000)).toBe('30s')
    })

    it('formats minutes', () => {
      expect(formatNsDuration(300_000_000_000)).toBe('5m')
    })

    it('formats minutes with remaining seconds', () => {
      expect(formatNsDuration(150_000_000_000)).toBe('2m 30s')
    })

    it('formats hours', () => {
      expect(formatNsDuration(7200_000_000_000)).toBe('2h')
    })

    it('formats hours with remaining minutes', () => {
      expect(formatNsDuration(5400_000_000_000)).toBe('1h 30m')
    })

    it('formats days', () => {
      expect(formatNsDuration(86400_000_000_000)).toBe('1d')
    })

    it('formats days with remaining hours', () => {
      expect(formatNsDuration(129600_000_000_000)).toBe('1d 12h')
    })
  })
})
