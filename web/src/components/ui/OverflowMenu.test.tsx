import { afterEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@/test/utils'
import { OverflowMenu } from './OverflowMenu'

function nearScrollerBottom(trigger: string) {
  vi.spyOn(HTMLElement.prototype, 'scrollHeight', 'get').mockReturnValue(120)
  vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockImplementation(function (this: HTMLElement) {
    if (this.dataset.testid === 'scroller') return { top: 0, bottom: 705 } as DOMRect
    if (this.querySelector(`button[aria-label="${trigger}"]`)) return { top: 660, bottom: 690 } as DOMRect
    return { top: 0, bottom: 0 } as DOMRect
  })
}

describe('OverflowMenu', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('opens upward when the scroll area has no room below it', () => {
    nearScrollerBottom('More actions')
    render(
      <div data-testid="scroller" style={{ overflowY: 'auto' }}>
        <OverflowMenu items={[{ label: 'Export', onSelect: vi.fn() }]} />
      </div>,
    )

    fireEvent.click(screen.getByRole('button', { name: 'More actions' }))

    expect(screen.getByRole('menu')).toHaveClass('bottom-full')
  })
})
