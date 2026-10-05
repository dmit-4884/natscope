import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import StreamStatsHeader from './StreamStatsHeader'

const { detail, liveState } = vi.hoisted(() => ({
  detail: { data: { state: { messages: 1234, bytes: 2048, consumer_count: 2 }, cluster: { name: 'C1' } } },
  liveState: { stats: { isConnected: true, msgPerSecond: 12 } },
}))

vi.mock('@/contexts/streams', () => ({ useStreamDetail: () => detail }))
vi.mock('@/contexts/live', () => ({
  useLiveStatsStore: (selector: (s: typeof liveState) => unknown) => selector(liveState),
}))

const LEVEL_WIDTHS = [600, 420, 300]
let containerWidth = 0

function labels() {
  const row = screen.getByTestId('stream-stats')
  return Array.from(row.children)
    .filter((el) => !el.hasAttribute('aria-hidden'))
    .map((el) => el.firstElementChild)
}

describe('StreamStatsHeader', () => {
  beforeEach(() => {
    vi.spyOn(HTMLElement.prototype, 'clientWidth', 'get').mockImplementation(function (this: HTMLElement) {
      return this.dataset.testid === 'stream-stats' ? containerWidth : 0
    })
    vi.spyOn(HTMLElement.prototype, 'offsetWidth', 'get').mockImplementation(function (this: HTMLElement) {
      return this.dataset.level === undefined ? 0 : LEVEL_WIDTHS[Number(this.dataset.level)]
    })
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('shows full labels when they fit', () => {
    containerWidth = 800
    render(<StreamStatsHeader streamName="ORDERS" connectionId="conn-1" />)

    expect(labels().map((el) => el?.textContent)).toEqual(['Messages', 'Consumers', 'Size', 'Messages/s', 'Bytes/s', 'Cluster'])
  })

  it('abbreviates labels and spells them out on hover when the full row does not fit', () => {
    containerWidth = 500
    render(<StreamStatsHeader streamName="ORDERS" connectionId="conn-1" />)

    const shown = labels()
    expect(shown.map((el) => el?.textContent)).toEqual(['Msgs', 'Cons', 'Size', 'Msg/s', 'B/s', 'Cluster'])
    expect(shown[3]?.tagName).toBe('ABBR')
    expect(shown[3]?.getAttribute('title')).toBe('Messages/s')
  })

  it('drops throughput and cluster when even the abbreviations do not fit', () => {
    containerWidth = 350
    render(<StreamStatsHeader streamName="ORDERS" connectionId="conn-1" />)

    expect(labels().map((el) => el?.textContent)).toEqual(['Msgs', 'Cons', 'Size', 'Msg/s'])
    expect(screen.getByTestId('stream-stats')).not.toHaveClass('flex-wrap')
  })

  it('refits before the next paint when its width changes', () => {
    const resized: Array<() => void> = []
    vi.stubGlobal(
      'ResizeObserver',
      class {
        constructor(callback: () => void) {
          resized.push(callback)
        }
        observe() {}
        disconnect() {}
      },
    )
    containerWidth = 800
    render(<StreamStatsHeader streamName="ORDERS" connectionId="conn-1" />)

    containerWidth = 500
    vi.spyOn(console, 'error').mockImplementation(() => {})
    resized.forEach((callback) => callback())

    expect(labels()[0]?.textContent).toBe('Msgs')
    vi.unstubAllGlobals()
  })

  it('wraps as a last resort', () => {
    containerWidth = 200
    render(<StreamStatsHeader streamName="ORDERS" connectionId="conn-1" />)

    expect(screen.getByTestId('stream-stats')).toHaveClass('flex-wrap')
  })
})
