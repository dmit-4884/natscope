import { describe, expect, it, vi } from 'vitest'
import { useLayoutEffect } from 'react'
import { render } from '@/test/utils'
import type { SelectedMessage } from '@/types/messages'
import UnifiedMessageViewer from './UnifiedMessageViewer'

vi.mock('@/contexts/mappings', () => ({
  useMappingItems: () => ({ data: [] }),
  useCreateMapping: () => ({ mutate: vi.fn(), isPending: false }),
}))

vi.mock('@/contexts/streams', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/contexts/streams')>()),
  useStreamDetail: () => ({ data: undefined }),
}))

vi.mock('./MessageConsumers', () => ({ MessageConsumers: () => null }))

function protoMessage(seq: number, name: string): SelectedMessage {
  return {
    id: `history-${seq}`,
    sequence: seq,
    subject: 'orders.created',
    timestamp: 0,
    data_base64: btoa('\u0006' + name),
    data_size: name.length + 1,
    content_type: 'binary',
    decoded: { name },
    decodedType: 'shop.Order',
  }
}

describe('UnifiedMessageViewer', () => {
  it('shows a newly selected message in its first frame, not the one before it', () => {
    const frames: string[] = []
    function Frame({ message }: { message: SelectedMessage }) {
      useLayoutEffect(() => {
        frames.push(document.body.textContent ?? '')
      })
      return <UnifiedMessageViewer streamName="ORDERS" connectionId="conn-1" selectedMessage={message} />
    }

    const view = render(<Frame message={protoMessage(1, 'picked')} />)
    frames.length = 0
    view.rerender(<Frame message={protoMessage(2, 'guessed')} />)

    expect(frames[0]).toContain('Message #2')
    expect(frames[0]).toContain('guessed')
    expect(frames[0]).not.toContain('picked')
  })

  it('shows the decoded payload in the first frame of a selection', () => {
    const frames: string[] = []
    function Frame({ message }: { message: SelectedMessage | null }) {
      useLayoutEffect(() => {
        frames.push(document.body.textContent ?? '')
      })
      return <UnifiedMessageViewer streamName="ORDERS" connectionId="conn-1" selectedMessage={message} />
    }

    const view = render(<Frame message={null} />)
    frames.length = 0
    view.rerender(<Frame message={protoMessage(3, 'decoded-at-once')} />)

    expect(frames[0]).toContain('Decoded Protobuf')
    expect(frames[0]).not.toContain('UTF-8')
  })
})
