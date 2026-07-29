import { useCallback, useEffect, useRef, useState } from 'react'

export interface UseCopyToClipboardResult {
  /** Copy a string to the clipboard. Returns true on success. */
  copy: (text: string) => Promise<boolean>
  /** True for `resetMs` after a successful copy — drive the "Copied!" UI off this. */
  isCopied: boolean
  /** Last error from the clipboard API (DOM exceptions, permission denial). */
  error: Error | null
}

/**
 * Copy-to-clipboard logic. Auto-resets `isCopied` after `resetMs` (timer
 * cancelled on unmount). Falls back to the deprecated `execCommand('copy')`
 * in non-secure contexts / cross-origin iframes where the clipboard API is
 * unavailable.
 */
export function useCopyToClipboard(resetMs: number = 1500): UseCopyToClipboardResult {
  const [isCopied, setIsCopied] = useState(false)
  const [error, setError] = useState<Error | null>(null)
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => {
    return () => {
      if (timerRef.current) clearTimeout(timerRef.current)
    }
  }, [])

  const copy = useCallback(
    async (text: string): Promise<boolean> => {
      try {
        if (navigator?.clipboard?.writeText) {
          await navigator.clipboard.writeText(text)
        } else {
          // Non-secure fallback: readonly + offscreen to avoid layout shift.
          const textarea = document.createElement('textarea')
          textarea.value = text
          textarea.setAttribute('readonly', '')
          textarea.style.position = 'fixed'
          textarea.style.top = '-9999px'
          textarea.style.left = '-9999px'
          document.body.appendChild(textarea)
          textarea.select()
          const ok = document.execCommand('copy')
          document.body.removeChild(textarea)
          if (!ok) throw new Error('document.execCommand("copy") failed')
        }
        setIsCopied(true)
        setError(null)
        if (timerRef.current) clearTimeout(timerRef.current)
        timerRef.current = setTimeout(() => setIsCopied(false), resetMs)
        return true
      } catch (e) {
        setError(e instanceof Error ? e : new Error(String(e)))
        setIsCopied(false)
        return false
      }
    },
    [resetMs],
  )

  return { copy, isCopied, error }
}
