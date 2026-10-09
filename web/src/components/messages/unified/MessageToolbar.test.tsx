import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@/test/utils'
import { EMPTY_FILTERS } from '../searchQuery'
import { MessageToolbar } from './MessageToolbar'

function renderToolbar(wsStatus: 'connecting' | 'connected') {
  render(
    <MessageToolbar
      mode="realtime"
      onModeChange={vi.fn()}
      filters={EMPTY_FILTERS}
      onFiltersChange={vi.fn()}
      onRemoveFilter={vi.fn()}
      onClearAllFilters={vi.fn()}
      showFiltersPanel={false}
      onToggleFiltersPanel={vi.fn()}
      availableSubjects={[]}
      messageCount={0}
      limit={50}
      onLimitChange={vi.fn()}
      onRefetch={vi.fn()}
      isFetching={false}
      liveLimit={100}
      onLiveLimitChange={vi.fn()}
      wsStatus={wsStatus}
      isPaused={false}
      onTogglePause={vi.fn()}
      onClearLive={vi.fn()}
      onMaxDisplayRateChange={vi.fn()}
      compareMode={false}
      onToggleCompareMode={vi.fn()}
      onOpenExport={vi.fn()}
    />,
  )
}

describe('MessageToolbar in realtime', () => {
  it('keeps the Pause button in place while the feed connects', () => {
    renderToolbar('connecting')
    expect(screen.getByRole('button', { name: /Pause/ })).toBeDisabled()
  })
})
