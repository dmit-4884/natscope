import { useRef } from 'react'

let counter = 0

/**
 * Stable React keys for editable list rows whose data model is a plain array
 * (no per-item id). Keys track rows by position through add/remove so React
 * preserves each row's DOM/input state (avoids focus jumps from index keys).
 *
 * Route the list's add/remove through the returned handlers so the key array
 * mutates in lock-step with the parent's value array. The render-time length
 * sync only covers external resets (e.g. loading a different record).
 */
export function useRowKeys(length: number): {
  keys: string[]
  registerAdd: () => void
  registerRemove: (index: number) => void
  reset: () => void
} {
  const keysRef = useRef<string[]>([])
  const keys = keysRef.current

  // Reconcile length for external changes (initial mount, record switch): grow
  // by appending fresh keys, shrink by dropping trailing ones. In-list removals
  // are handled precisely by registerRemove before the re-render arrives here.
  while (keys.length < length) keys.push(`row-${counter++}`)
  if (keys.length > length) keys.length = length

  return {
    keys,
    registerAdd: () => {
      keys.push(`row-${counter++}`)
    },
    registerRemove: (index: number) => {
      keys.splice(index, 1)
    },
    reset: () => {
      keys.length = 0
    },
  }
}
