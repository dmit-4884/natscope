import { describe, expect, it } from 'vitest'
import { useLayoutEffect } from 'react'
import { render } from '@/test/utils'
import JsonTreeViewer from './JsonTreeViewer'

describe('JsonTreeViewer', () => {
  it('shows new data expanded in its first frame', () => {
    const frames: string[] = []
    function Frame({ data }: { data: unknown }) {
      useLayoutEffect(() => {
        frames.push(document.body.textContent ?? '')
      })
      return <JsonTreeViewer data={data} title="Decoded" />
    }

    const view = render(<Frame data={{ name: 'first' }} />)
    frames.length = 0
    view.rerender(<Frame data={{ name: 'second', tags: ['inner-tag'] }} />)

    expect(frames[0]).toContain('inner-tag')
  })
})
