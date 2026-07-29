import { useState, useRef, useEffect } from 'react'
import type { ReactNode } from 'react'
import { isMacPlatform } from '@/utils/platform'

interface TooltipProps {
  content: string
  children: ReactNode
  delay?: number
  position?: 'top' | 'bottom' | 'left' | 'right'
  /** Keyboard shortcut to display alongside content */
  shortcut?: string
}

/**
 * Format keyboard shortcut for display
 * Converts mod+ to platform-specific symbol
 */
function formatShortcut(shortcut: string): string {
  const isMac = isMacPlatform()
  return shortcut
    .replace(/mod\+/gi, isMac ? '\u2318' : 'Ctrl+')
    .replace(/shift\+/gi, isMac ? '\u21E7' : 'Shift+')
    .replace(/alt\+/gi, isMac ? '\u2325' : 'Alt+')
    .replace(/ctrl\+/gi, isMac ? '\u2303' : 'Ctrl+')
}

export default function Tooltip({
  content,
  children,
  delay = 300,
  position = 'top',
  shortcut,
}: TooltipProps) {
  const [isVisible, setIsVisible] = useState(false)
  const [coords, setCoords] = useState({ x: 0, y: 0 })
  const timeoutRef = useRef<number | null>(null)
  const elementRef = useRef<HTMLDivElement>(null)

  const showTooltip = () => {
    timeoutRef.current = window.setTimeout(() => {
      if (elementRef.current) {
        const rect = elementRef.current.getBoundingClientRect()
        setCoords({
          x: rect.left + rect.width / 2,
          y: position === 'bottom' ? rect.bottom : rect.top,
        })
      }
      setIsVisible(true)
    }, delay)
  }

  const hideTooltip = () => {
    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current)
      timeoutRef.current = null
    }
    setIsVisible(false)
  }

  useEffect(() => {
    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current)
      }
    }
  }, [])

  if (!content && !shortcut) {
    return <>{children}</>
  }

  const formattedShortcut = shortcut ? formatShortcut(shortcut) : null

  return (
    <>
      <span
        ref={elementRef}
        onMouseEnter={showTooltip}
        onMouseLeave={hideTooltip}
        onFocus={showTooltip}
        onBlur={hideTooltip}
        className="inline-flex"
      >
        {children}
      </span>
      {isVisible && (
        <div
          className="fixed z-50 px-2 py-1 text-xs text-content-inverse bg-surface-inverse rounded shadow-lg pointer-events-none max-w-sm flex items-center gap-2"
          style={{
            left: coords.x,
            top: position === 'bottom' ? coords.y + 8 : coords.y - 8,
            transform: `translateX(-50%) ${position === 'bottom' ? '' : 'translateY(-100%)'}`,
          }}
          role="tooltip"
        >
          <span>{content}</span>
          {formattedShortcut && (
            <kbd className="px-1.5 py-0.5 text-2xs bg-gray-700 rounded font-mono text-gray-300">
              {formattedShortcut}
            </kbd>
          )}
        </div>
      )}
    </>
  )
}
