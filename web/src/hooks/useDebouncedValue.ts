import { useEffect, useState } from 'react'

/**
 * Returns a value that lags behind `value` by `delay` ms. Re-types between
 * frames (or any high-frequency state change) collapse into a single trailing
 * update — useful for search inputs feeding network queries.
 */
export function useDebouncedValue<T>(value: T, delay: number = 300): T {
  const [debouncedValue, setDebouncedValue] = useState<T>(value)

  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedValue(value)
    }, delay)

    return () => {
      clearTimeout(timer)
    }
  }, [value, delay])

  return debouncedValue
}
