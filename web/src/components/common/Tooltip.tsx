import { useState, useRef, useEffect, useLayoutEffect, useCallback } from 'react'
import type { ReactNode } from 'react'
import { createPortal } from 'react-dom'
import { isMacPlatform } from '@/utils/platform'
import { placeTooltip, type TooltipSide } from './tooltipPlacement'

interface TooltipProps {
  content: string
  children: ReactNode
  delay?: number
  position?: TooltipSide
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
    .replace(/mod\+/gi, isMac ? '⌘' : 'Ctrl+')
    .replace(/shift\+/gi, isMac ? '⇧' : 'Shift+')
    .replace(/alt\+/gi, isMac ? '⌥' : 'Alt+')
    .replace(/ctrl\+/gi, isMac ? '⌃' : 'Ctrl+')
}

export default function Tooltip({
  content,
  children,
  delay = 300,
  position = 'top',
  shortcut,
}: TooltipProps) {
  const [isVisible, setIsVisible] = useState(false)
  const [place, setPlace] = useState<{ left: number; top: number } | null>(null)
  const timeoutRef = useRef<number | null>(null)
  const triggerRef = useRef<HTMLSpanElement>(null)
  const tipRef = useRef<HTMLDivElement>(null)

  const showTooltip = () => {
    timeoutRef.current = window.setTimeout(() => setIsVisible(true), delay)
  }

  const hideTooltip = useCallback(() => {
    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current)
      timeoutRef.current = null
    }
    setIsVisible(false)
    setPlace(null)
  }, [])

  useLayoutEffect(() => {
    if (!isVisible || !triggerRef.current || !tipRef.current) return
    const tip = tipRef.current.getBoundingClientRect()
    setPlace(
      placeTooltip(
        triggerRef.current.getBoundingClientRect(),
        { width: tip.width, height: tip.height },
        { width: window.innerWidth, height: window.innerHeight },
        position,
      ),
    )
  }, [isVisible, position, content, shortcut])

  useEffect(() => {
    if (!isVisible) return
    window.addEventListener('scroll', hideTooltip, true)
    window.addEventListener('resize', hideTooltip)
    return () => {
      window.removeEventListener('scroll', hideTooltip, true)
      window.removeEventListener('resize', hideTooltip)
    }
  }, [isVisible, hideTooltip])

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
        ref={triggerRef}
        onMouseEnter={showTooltip}
        onMouseLeave={hideTooltip}
        onFocus={showTooltip}
        onBlur={hideTooltip}
        className="inline-flex"
      >
        {children}
      </span>
      {isVisible &&
        createPortal(
          <div
            ref={tipRef}
            className="fixed z-tooltip w-max max-w-[min(20rem,calc(100vw-16px))] px-2.5 py-1.5 text-xs leading-snug text-content-inverse bg-surface-inverse rounded-md shadow-lg pointer-events-none flex items-center gap-2 whitespace-normal break-words"
            style={place ? { left: place.left, top: place.top } : { left: 0, top: 0, visibility: 'hidden' }}
            role="tooltip"
          >
            <span>{content}</span>
            {formattedShortcut && (
              <kbd className="px-1.5 py-0.5 text-2xs bg-gray-700 rounded font-mono text-gray-300">
                {formattedShortcut}
              </kbd>
            )}
          </div>,
          document.body,
        )}
    </>
  )
}
