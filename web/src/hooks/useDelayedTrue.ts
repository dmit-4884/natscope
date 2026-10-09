import { useEffect, useState } from 'react'

export function useDelayedTrue(value: boolean, delayMs: number): boolean {
  const [held, setHeld] = useState(false)
  useEffect(() => {
    if (!value) {
      setHeld(false)
      return
    }
    const timer = setTimeout(() => setHeld(true), delayMs)
    return () => clearTimeout(timer)
  }, [value, delayMs])
  return value && held
}
