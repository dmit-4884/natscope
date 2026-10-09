import { describe, expect, it } from 'vitest'
import { useLayoutEffect, type ComponentProps } from 'react'
import { render } from '@/test/utils'
import PayloadViewer from './PayloadViewer'

describe('PayloadViewer', () => {
  it('shows a JSON payload in its first frame after a decoded one', () => {
    const frames: string[] = []
    function Frame(props: ComponentProps<typeof PayloadViewer>) {
      useLayoutEffect(() => {
        frames.push(document.body.textContent ?? '')
      })
      return <PayloadViewer {...props} />
    }

    const view = render(<Frame rawData={btoa('\u0008\u0001')} decodedData={{ name: 'picked' }} defaultMode="decoded" />)
    frames.length = 0
    view.rerender(<Frame rawData={btoa('{"order":"o-1"}')} decodedData={null} jsonData={{ order: 'o-1' }} defaultMode="json" />)

    expect(frames[0]).toContain('JSON Data')
    expect(frames[0]).toContain('o-1')
  })
})
