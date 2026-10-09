import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render } from '@testing-library/react'
import type { Message } from '@/types/nats'
import type { LiveMessage } from './messageListUtils'

const scrollToIndex = vi.fn()
vi.mock('@tanstack/react-virtual', () => ({
  useVirtualizer: ({ count }: { count: number }) => ({
    getVirtualItems: () =>
      Array.from({ length: count }, (_, index) => ({ index, start: index * 52, size: 52, key: index })),
    getTotalSize: () => count * 52,
    scrollToIndex,
  }),
}))

import { MessageVirtualTable } from './MessageVirtualTable'

function backwardMessages(): Message[] {
  return Array.from({ length: 20 }, (_, i) => {
    const sequence = 20 - i
    return {
      sequence,
      subject: `TEST.subject.${sequence}`,
      timestamp: 1_700_000_000_000 + sequence,
      data_base64: '',
      data_size: 10,
      content_type: 'json' as const,
    }
  })
}

function renderTable(selectedMessageId: string | null) {
  const messages = backwardMessages()
  return render(
    <MessageVirtualTable
      messages={messages}
      mode="history"
      selectedMessageId={selectedMessageId}
      newMessageIds={new Set()}
      compareMode={false}
      isCompareSelected={() => false}
      rowHeight={52}
      cellPadding="px-3 py-2"
      streamName="TEST"
      connectionId={null}
      timestampFormat="relative"
      autoScrollRef={{ current: false }}
      onSelectHistory={() => {}}
      onSelectLive={() => {}}
      onCompareSelect={() => {}}
    />,
  )
}

describe('MessageVirtualTable selection scroll-into-view', () => {
  beforeEach(() => scrollToIndex.mockClear())

  it('scrolls the selected row into view when selection changes to a loaded message', () => {
    const { rerender } = renderTable(null)
    expect(scrollToIndex).not.toHaveBeenCalled()

    rerender(
      <MessageVirtualTable
        messages={backwardMessages()}
        mode="history"
        selectedMessageId="history-11"
        newMessageIds={new Set()}
        compareMode={false}
        isCompareSelected={() => false}
        rowHeight={52}
        cellPadding="px-3 py-2"
        streamName="TEST"
        connectionId={null}
        timestampFormat="relative"
        autoScrollRef={{ current: false }}
        onSelectHistory={() => {}}
        onSelectLive={() => {}}
        onCompareSelect={() => {}}
      />,
    )

    expect(scrollToIndex).toHaveBeenCalledWith(9, expect.objectContaining({ align: 'auto' }))
  })

  it('does not scroll when the selected message is not in the loaded page', () => {
    const { rerender } = renderTable(null)
    scrollToIndex.mockClear()

    // Navigation stepped past the loaded window — no row to scroll to.
    rerender(
      <MessageVirtualTable
        messages={backwardMessages()}
        mode="history"
        selectedMessageId="history-9999"
        newMessageIds={new Set()}
        compareMode={false}
        isCompareSelected={() => false}
        rowHeight={52}
        cellPadding="px-3 py-2"
        streamName="TEST"
        connectionId={null}
        timestampFormat="relative"
        autoScrollRef={{ current: false }}
        onSelectHistory={() => {}}
        onSelectLive={() => {}}
        onCompareSelect={() => {}}
      />,
    )

    expect(scrollToIndex).not.toHaveBeenCalled()
  })
})

function liveMessages(ids: number[]): LiveMessage[] {
  return ids.map((n) => ({
    id: `live-${n}`,
    stream_name: 'TEST',
    subject: `TEST.subject.${n}`,
    timestamp: 1_700_000_000_000 + n,
    data_base64: '',
    data_size: 10,
    content_type: 'json' as const,
  }))
}

const liveTable = (messages: LiveMessage[], autoScrollRef: { current: boolean }) => (
  <MessageVirtualTable
    messages={messages}
    mode="realtime"
    selectedMessageId={null}
    newMessageIds={new Set()}
    compareMode={false}
    isCompareSelected={() => false}
    rowHeight={52}
    cellPadding="px-3 py-2"
    streamName="TEST"
    connectionId={null}
    timestampFormat="relative"
    autoScrollRef={autoScrollRef}
    onSelectHistory={() => {}}
    onSelectLive={() => {}}
    onCompareSelect={() => {}}
  />
)

describe('MessageVirtualTable rows', () => {
  it('move to their new place at once instead of sliding over each other', () => {
    const { getAllByRole } = renderTable(null)
    const row = getAllByRole('row')[1]
    expect(row.className).not.toMatch(/transition-all/)
  })
})

describe('MessageVirtualTable live feed scrolled away from the top', () => {
  it('keeps the rows in view still when new messages arrive above them', () => {
    const autoScrollRef = { current: false }
    const view = render(liveTable(liveMessages([3, 2, 1]), autoScrollRef))
    const grid = view.getByRole('grid')
    grid.scrollTop = 440

    view.rerender(liveTable(liveMessages([5, 4, 3, 2, 1]), autoScrollRef))

    expect(grid.scrollTop).toBe(440 + 2 * 52)
  })
})
