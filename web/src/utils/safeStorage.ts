/**
 * Safe localStorage wrappers. Direct access throws SecurityError in Safari
 * Private Mode / cross-origin iframes; a throw in a useState lazy initializer
 * escapes error boundaries and crashes before paint. Use for any read/write
 * before an ErrorBoundary mounts. Failures are silent on purpose.
 */

export function safeGetItem(key: string): string | null {
  try {
    return localStorage.getItem(key)
  } catch {
    return null
  }
}

export function safeSetItem(key: string, value: string): boolean {
  try {
    localStorage.setItem(key, value)
    return true
  } catch {
    return false
  }
}

export function safeRemoveItem(key: string): boolean {
  try {
    localStorage.removeItem(key)
    return true
  } catch {
    return false
  }
}
