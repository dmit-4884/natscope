import { describe, expect, it } from 'vitest'
import { useLayoutEffect, type ComponentProps } from 'react'
import { fireEvent, render, screen } from '@/test/utils'
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

  it('opens the next message in its own view, not the one picked for the previous message', () => {
    const frames: string[] = []
    function Frame(props: ComponentProps<typeof PayloadViewer>) {
      useLayoutEffect(() => {
        frames.push(document.body.textContent ?? '')
      })
      return <PayloadViewer {...props} />
    }

    const view = render(<Frame rawData={btoa('{"order":"o-1"}')} jsonData={{ order: 'o-1' }} defaultMode="json" />)
    fireEvent.click(screen.getByRole('tab', { name: /Hex/ }))
    expect(document.body.textContent).toContain('Hex Dump')
    frames.length = 0
    view.rerender(<Frame rawData={btoa('{"order":"o-2"}')} jsonData={{ order: 'o-2' }} defaultMode="json" />)

    expect(frames[0]).not.toContain('Hex Dump')
    expect(frames[0]).toContain('JSON Data')
  })
})
