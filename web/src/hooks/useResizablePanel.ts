import { useCallback, useEffect, useRef, useState } from 'react'
import { safeGetItem, safeSetItem } from '@/utils/safeStorage'

const RIGHT_PANEL_WIDTH_KEY = 'nats_right_panel_width'
const DEFAULT_RIGHT_PANEL_PCT = 50
const MIN_PANEL_PCT = 20
const MAX_PANEL_PCT = 80
const RESIZE_STEP_PCT = 2

const clamp = (pct: number) => Math.min(Math.max(pct, MIN_PANEL_PCT), MAX_PANEL_PCT)

export function useResizablePanel() {
  const [rightPanelPct, setRightPanelPct] = useState(() => {
    const saved = safeGetItem(RIGHT_PANEL_WIDTH_KEY)
    return saved ? Number(saved) : DEFAULT_RIGHT_PANEL_PCT
  })
  const isDragging = useRef(false)
  const containerRef = useRef<HTMLDivElement>(null)
  const latestPctRef = useRef(rightPanelPct)

  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      if (!isDragging.current || !containerRef.current) return
      const rect = containerRef.current.getBoundingClientRect()
      const clamped = clamp(((rect.right - e.clientX) / rect.width) * 100)
      latestPctRef.current = clamped
      setRightPanelPct(clamped)
    }
    const handleMouseUp = () => {
      if (isDragging.current) {
        isDragging.current = false
        document.body.style.cursor = ''
        document.body.style.userSelect = ''
        safeSetItem(RIGHT_PANEL_WIDTH_KEY, String(Math.round(latestPctRef.current)))
      }
    }
    document.addEventListener('mousemove', handleMouseMove)
    document.addEventListener('mouseup', handleMouseUp)
    return () => {
      document.removeEventListener('mousemove', handleMouseMove)
      document.removeEventListener('mouseup', handleMouseUp)
    }
  }, [])

  const onMouseDown = useCallback((e: React.MouseEvent) => {
    e.preventDefault()
    isDragging.current = true
    document.body.style.cursor = 'col-resize'
    document.body.style.userSelect = 'none'
  }, [])

  const onKeyDown = useCallback((e: React.KeyboardEvent) => {
    const step = e.key === 'ArrowLeft' ? RESIZE_STEP_PCT : e.key === 'ArrowRight' ? -RESIZE_STEP_PCT : 0
    if (step === 0) return
    e.preventDefault()
    setRightPanelPct((prev) => {
      const next = clamp(prev + step)
      latestPctRef.current = next
      safeSetItem(RIGHT_PANEL_WIDTH_KEY, String(Math.round(next)))
      return next
    })
  }, [])

  return {
    rightPanelPct,
    containerRef,
    separatorProps: {
      role: 'separator',
      'aria-orientation': 'vertical',
      'aria-label': 'Resize details panel',
      'aria-valuenow': Math.round(rightPanelPct),
      'aria-valuemin': MIN_PANEL_PCT,
      'aria-valuemax': MAX_PANEL_PCT,
      tabIndex: 0,
      onMouseDown,
      onKeyDown,
    } as const,
  }
}
