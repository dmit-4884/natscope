// Logger: debug/info only in dev; warn/error always.
const isDev = import.meta.env.DEV

export const logger = {
  /** Dev-only */
  debug: (...args: unknown[]) => {
    if (isDev) console.log('[DEBUG]', ...args)
  },

  /** Dev-only */
  info: (...args: unknown[]) => {
    if (isDev) console.log('[INFO]', ...args)
  },

  /** Always shown */
  warn: (...args: unknown[]) => {
    console.warn('[WARN]', ...args)
  },

  /** Always shown */
  error: (...args: unknown[]) => {
    console.error('[ERROR]', ...args)
  },
}
