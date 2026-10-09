import { useLayoutEffect, useState, type RefObject } from 'react'

export interface PopupPlacement {
  side: 'below' | 'above'
  maxHeight?: number
}

interface Bounds {
  top: number
  bottom: number
}

const GAP_PX = 4
const MIN_HEIGHT_PX = 80
const CLIPPING = new Set(['auto', 'scroll', 'hidden', 'clip'])

export function clipBounds(anchor: HTMLElement): Bounds {
  let top = 0
  let bottom = window.innerHeight
  for (let node = anchor.parentElement; node && node !== document.body; node = node.parentElement) {
    if (!CLIPPING.has(getComputedStyle(node).overflowY)) continue
    const rect = node.getBoundingClientRect()
    top = Math.max(top, rect.top)
    bottom = Math.min(bottom, rect.bottom)
  }
  return { top, bottom }
}

export function placePopup(anchor: DOMRect, bounds: Bounds, wanted: number): PopupPlacement {
  const below = bounds.bottom - anchor.bottom - GAP_PX
  if (below >= wanted) return { side: 'below' }
  const above = anchor.top - bounds.top - GAP_PX
  const side = above > below ? 'above' : 'below'
  const room = side === 'above' ? above : below
  return room < wanted ? { side, maxHeight: Math.max(room, MIN_HEIGHT_PX) } : { side }
}

export function scrollWithin(list: HTMLElement, item: HTMLElement): void {
  const listRect = list.getBoundingClientRect()
  const itemRect = item.getBoundingClientRect()
  if (itemRect.top < listRect.top) list.scrollTop -= listRect.top - itemRect.top
  else if (itemRect.bottom > listRect.bottom) list.scrollTop += itemRect.bottom - listRect.bottom
}

export function usePopupPlacement(
  open: boolean,
  anchorRef: RefObject<HTMLElement | null>,
  popupRef: RefObject<HTMLElement | null>,
): PopupPlacement | null {
  const [placement, setPlacement] = useState<PopupPlacement | null>(null)
  useLayoutEffect(() => {
    const anchor = anchorRef.current
    const popup = popupRef.current
    if (!open || !anchor || !popup) {
      setPlacement(null)
      return
    }
    const cap = parseFloat(getComputedStyle(popup).maxHeight)
    const wanted = Math.min(popup.scrollHeight, Number.isNaN(cap) ? Infinity : cap)
    setPlacement(placePopup(anchor.getBoundingClientRect(), clipBounds(anchor), wanted))
  }, [open, anchorRef, popupRef])
  return placement
}
