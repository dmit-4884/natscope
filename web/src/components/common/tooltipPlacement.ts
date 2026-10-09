export type TooltipSide = 'top' | 'bottom' | 'left' | 'right'

interface Box {
  left: number
  top: number
  width: number
  height: number
  right: number
  bottom: number
}

interface Size {
  width: number
  height: number
}

const GAP = 8
const MARGIN = 8

const clamp = (value: number, min: number, max: number) => Math.max(min, Math.min(value, max))

export function placeTooltip(trigger: Box, tip: Size, viewport: Size, side: TooltipSide): { left: number; top: number } {
  if (side === 'left' || side === 'right') {
    const roomLeft = trigger.left - GAP - MARGIN
    const roomRight = viewport.width - trigger.right - GAP - MARGIN
    const onRight = side === 'right' ? tip.width <= roomRight || roomRight >= roomLeft : tip.width > roomLeft && roomRight > roomLeft
    const left = onRight ? trigger.right + GAP : trigger.left - GAP - tip.width
    const top = trigger.top + trigger.height / 2 - tip.height / 2
    return {
      left: clamp(left, MARGIN, viewport.width - tip.width - MARGIN),
      top: clamp(top, MARGIN, viewport.height - tip.height - MARGIN),
    }
  }
  const roomAbove = trigger.top - GAP - MARGIN
  const roomBelow = viewport.height - trigger.bottom - GAP - MARGIN
  const below = side === 'bottom' ? tip.height <= roomBelow || roomBelow >= roomAbove : tip.height > roomAbove && roomBelow > roomAbove
  const top = below ? trigger.bottom + GAP : trigger.top - GAP - tip.height
  const left = trigger.left + trigger.width / 2 - tip.width / 2
  return {
    left: clamp(left, MARGIN, viewport.width - tip.width - MARGIN),
    top: clamp(top, MARGIN, viewport.height - tip.height - MARGIN),
  }
}
