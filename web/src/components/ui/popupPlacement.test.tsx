import { describe, expect, it } from 'vitest'
import { clipBounds, placePopup, scrollWithin } from './popupPlacement'

const rect = (top: number, bottom: number) => ({ top, bottom }) as DOMRect

describe('placePopup', () => {
  it('opens below when the popup fits there', () => {
    expect(placePopup(rect(100, 130), { top: 0, bottom: 800 }, 200)).toEqual({ side: 'below' })
  })

  it('opens above when there is no room below and more room above', () => {
    expect(placePopup(rect(660, 690), { top: 0, bottom: 705 }, 66)).toEqual({ side: 'above' })
  })

  it('keeps a usable height when neither side has room', () => {
    expect(placePopup(rect(10, 40), { top: 0, bottom: 50 }, 240)).toEqual({ side: 'below', maxHeight: 80 })
  })

  it('shrinks to the room on the side it opens on', () => {
    expect(placePopup(rect(100, 130), { top: 0, bottom: 200 }, 240)).toEqual({ side: 'above', maxHeight: 96 })
  })
})

describe('clipBounds', () => {
  it('stops at the nearest scrolling ancestor', () => {
    const scroller = document.createElement('div')
    scroller.style.overflowY = 'auto'
    scroller.getBoundingClientRect = () => rect(50, 705)
    const anchor = document.createElement('button')
    scroller.appendChild(anchor)
    document.body.appendChild(scroller)

    expect(clipBounds(anchor)).toEqual({ top: 50, bottom: 705 })
    scroller.remove()
  })
})

describe('scrollWithin', () => {
  it('scrolls only the list to show an item below its view', () => {
    const list = document.createElement('ul')
    list.getBoundingClientRect = () => rect(0, 100)
    const item = document.createElement('li')
    item.getBoundingClientRect = () => rect(180, 210)
    list.appendChild(item)

    scrollWithin(list, item)

    expect(list.scrollTop).toBe(110)
  })
})
