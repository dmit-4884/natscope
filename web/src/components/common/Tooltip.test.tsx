import { afterEach, describe, expect, it, vi } from 'vitest'
import { act, fireEvent, render, screen } from '@testing-library/react'
import Tooltip from './Tooltip'
import { placeTooltip } from './tooltipPlacement'

const viewport = { width: 1000, height: 700 }
const rect = (left: number, top: number, width: number, height: number) => ({ left, top, width, height, right: left + width, bottom: top + height })

describe('placeTooltip', () => {
  it('centres the tip above its trigger', () => {
    expect(placeTooltip(rect(400, 300, 100, 20), { width: 200, height: 40 }, viewport, 'top')).toEqual({ left: 350, top: 252 })
  })

  it('keeps the tip inside the window next to an edge', () => {
    const atRight = placeTooltip(rect(950, 300, 40, 20), { width: 240, height: 40 }, viewport, 'top')
    expect(atRight.left + 240).toBeLessThanOrEqual(viewport.width - 8)
    const atLeft = placeTooltip(rect(0, 300, 20, 20), { width: 240, height: 40 }, viewport, 'top')
    expect(atLeft.left).toBe(8)
  })

  it('turns to the side that has room', () => {
    expect(placeTooltip(rect(400, 10, 100, 20), { width: 200, height: 40 }, viewport, 'top').top).toBe(38)
    expect(placeTooltip(rect(400, 680, 100, 20), { width: 200, height: 40 }, viewport, 'bottom').top).toBe(632)
    expect(placeTooltip(rect(960, 300, 30, 20), { width: 120, height: 30 }, viewport, 'right').left).toBe(832)
  })
})

describe('Tooltip', () => {
  afterEach(() => {
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  it('opens in the page body, at its full width and inside the window', () => {
    vi.useFakeTimers()
    vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockImplementation(function (this: HTMLElement) {
      const box = this.getAttribute('role') === 'tooltip' ? rect(0, 0, 240, 48) : rect(window.innerWidth - 30, 300, 24, 24)
      return { ...box, x: box.left, y: box.top, toJSON: () => box } as DOMRect
    })
    render(
      <Tooltip content="Rank every message type by how well this payload decodes as it">
        <button type="button">Detect type</button>
      </Tooltip>,
    )

    fireEvent.mouseEnter(screen.getByRole('button', { name: 'Detect type' }).parentElement!)
    act(() => vi.advanceTimersByTime(300))

    const tip = screen.getByRole('tooltip')
    expect(tip.parentElement).toBe(document.body)
    expect(tip.className).toContain('w-max')
    expect(parseFloat(tip.style.left) + 240).toBeLessThanOrEqual(window.innerWidth - 8)
    expect(tip.style.visibility).not.toBe('hidden')
  })
})
