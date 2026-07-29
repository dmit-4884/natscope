import { Dropdown, RefreshIcon, ClipboardIcon, DownloadIcon } from '@/components/ui'
import Tooltip from '@/components/common/Tooltip'
import { plural } from '@/utils/plural'
import type { FilterValues } from '../AdvancedFilters'
import FilterChips from '../FilterChips'
import AdvancedFilters from '../AdvancedFilters'
import { LIVE_MESSAGE_LIMITS, type LiveMessageLimit, type ViewMode, type WsStatus } from './messageListUtils'

interface Props {
  mode: ViewMode
  onModeChange: (mode: ViewMode) => void
  filters: FilterValues
  onFiltersChange: (filters: FilterValues) => void
  onRemoveFilter: (key: keyof FilterValues) => void
  onClearAllFilters: () => void
  showFiltersPanel: boolean
  onToggleFiltersPanel: () => void
  availableSubjects: string[]
  messageCount: number

  // History-only
  limit: number
  onLimitChange: (limit: number) => void
  onRefetch: () => void
  isFetching: boolean

  // Realtime-only
  liveLimit: LiveMessageLimit
  onLiveLimitChange: (limit: LiveMessageLimit) => void
  wsStatus: WsStatus
  isPaused: boolean
  onTogglePause: () => void
  onClearLive: () => void
  maxDisplayRate?: number
  onMaxDisplayRateChange: (rate: number) => void

  // Compare
  compareMode: boolean
  onToggleCompareMode: () => void

  onOpenExport: () => void
}

export function MessageToolbar(props: Props) {
  const {
    mode,
    onModeChange,
    filters,
    onFiltersChange,
    onRemoveFilter,
    onClearAllFilters,
    showFiltersPanel,
    onToggleFiltersPanel,
    availableSubjects,
    messageCount,
    limit,
    onLimitChange,
    onRefetch,
    isFetching,
    liveLimit,
    onLiveLimitChange,
    wsStatus,
    isPaused,
    onTogglePause,
    onClearLive,
    maxDisplayRate,
    onMaxDisplayRateChange,
    compareMode,
    onToggleCompareMode,
    onOpenExport,
  } = props

  const hasActiveFilters = !!(filters.subject || filters.startSequence || filters.startDate || filters.contentFilter)

  return (
    <div className="px-4 py-2 bg-surface-secondary border-b">
      <div className="flex items-center justify-between gap-4">
        {/* Min-width keeps the counter visible; otherwise the right button row collapses the parent to ~0px. */}
        <div className="flex items-center gap-3 flex-1 min-w-[140px]">
          <div className="relative shrink-0">
            <Tooltip content="Filters">
              <button
                onClick={onToggleFiltersPanel}
                className={`p-2 rounded-md border transition-colors ${
                  showFiltersPanel || hasActiveFilters
                    ? 'bg-accent-light border-blue-200 text-accent'
                    : 'bg-surface-primary border-border-strong text-content-secondary hover:bg-surface-secondary'
                }`}
                aria-label="Filters"
                aria-expanded={showFiltersPanel}
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2.586a1 1 0 01-.293.707l-6.414 6.414a1 1 0 00-.293.707V17l-4 4v-6.586a1 1 0 00-.293-.707L3.293 7.293A1 1 0 013 6.586V4z"
                  />
                </svg>
              </button>
            </Tooltip>
            {showFiltersPanel && (
              <AdvancedFilters
                filters={filters}
                availableSubjects={availableSubjects}
                onFiltersChange={onFiltersChange}
                onClose={onToggleFiltersPanel}
              />
            )}
          </div>

          <span className="text-sm text-content-secondary whitespace-nowrap shrink-0">{plural(messageCount, 'message')}</span>

          <div className="min-w-0 flex-1 overflow-hidden">
            <FilterChips filters={filters} onRemoveFilter={onRemoveFilter} onClearAll={onClearAllFilters} />
          </div>
        </div>

        <div className="flex items-center gap-2 shrink-0">
          {mode === 'history' && (
            <>
              <Dropdown
                size="sm"
                value={String(limit)}
                onChange={(v) => onLimitChange(Number(v))}
                options={[
                  { value: '25', label: '25' },
                  { value: '50', label: '50' },
                  { value: '100', label: '100' },
                  { value: '250', label: '250' },
                  { value: '500', label: '500' },
                ]}
              />
              <Tooltip content="Refresh">
                <button
                  onClick={onRefetch}
                  disabled={isFetching}
                  className={`p-2 rounded-md transition-colors ${
                    isFetching ? 'text-content-muted bg-surface-tertiary' : 'text-accent hover:bg-accent-light'
                  }`}
                  aria-label="Refresh"
                >
                  <RefreshIcon className={`w-4 h-4 ${isFetching ? 'animate-spin' : ''}`} />
                </button>
              </Tooltip>
            </>
          )}

          {mode === 'realtime' && (
            <>
              <Dropdown
                size="sm"
                value={String(liveLimit)}
                onChange={(v) => onLiveLimitChange(Number(v) as LiveMessageLimit)}
                options={LIVE_MESSAGE_LIMITS.map((l) => ({ value: String(l), label: String(l) }))}
              />
              {wsStatus === 'connected' && (
                <button
                  onClick={onTogglePause}
                  className={`px-3 py-1.5 text-xs font-medium rounded-md transition-colors flex items-center gap-1 ${
                    isPaused
                      ? 'bg-status-success-light text-green-700 hover:bg-green-200'
                      : 'bg-status-warning-light text-amber-700 hover:bg-amber-200'
                  }`}
                >
                  {isPaused ? (
                    <>
                      <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 24 24">
                        <path d="M8 5v14l11-7z" />
                      </svg>
                      Resume
                    </>
                  ) : (
                    <>
                      <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 24 24">
                        <path d="M6 4h4v16H6zm8 0h4v16h-4z" />
                      </svg>
                      Pause
                    </>
                  )}
                </button>
              )}
              <button
                onClick={onClearLive}
                className="px-3 py-1.5 text-xs font-medium text-content-secondary bg-surface-tertiary hover:bg-surface-hover rounded-md transition-colors"
              >
                Clear
              </button>
              <Dropdown
                size="sm"
                value={String(maxDisplayRate ?? 0)}
                onChange={(v) => onMaxDisplayRateChange(Number(v))}
                options={[
                  { value: '0', label: 'No limit' },
                  { value: '1', label: '1 msg/s' },
                  { value: '5', label: '5 msg/s' },
                  { value: '10', label: '10 msg/s' },
                  { value: '25', label: '25 msg/s' },
                  { value: '50', label: '50 msg/s' },
                ]}
              />
            </>
          )}

          {mode === 'history' && (
            <>
              <button
                onClick={onToggleCompareMode}
                className={`px-2.5 py-1.5 text-xs font-medium rounded-md border transition-colors flex items-center gap-1.5 ${
                  compareMode
                    ? 'bg-accent-light border-blue-200 text-accent'
                    : 'bg-surface-primary border-border-strong text-content-secondary hover:bg-surface-secondary'
                }`}
              >
                <ClipboardIcon className="w-3.5 h-3.5" />
                Diff
              </button>
              <button
                onClick={onOpenExport}
                className="px-2.5 py-1.5 text-xs font-medium rounded-md border bg-surface-primary border-border-strong text-content-secondary hover:bg-surface-secondary transition-colors flex items-center gap-1.5"
              >
                <DownloadIcon className="w-3.5 h-3.5" />
                Export
              </button>
            </>
          )}

          <div className="flex rounded-md overflow-hidden border border-border-strong">
            <button
              onClick={() => onModeChange('realtime')}
              className={`px-3 py-1.5 text-sm font-medium transition-colors flex items-center gap-1.5 ${
                mode === 'realtime' ? 'bg-accent text-content-inverse' : 'bg-surface-primary text-gray-700 hover:bg-surface-secondary'
              }`}
            >
              {mode === 'realtime' && wsStatus === 'connected' && (
                <>
                  <span aria-hidden="true" className="w-2 h-2 rounded-full bg-surface-primary animate-pulse" />
                  <span className="sr-only">Connected</span>
                </>
              )}
              Realtime
            </button>
            <button
              onClick={() => onModeChange('history')}
              className={`px-3 py-1.5 text-sm font-medium transition-colors ${
                mode === 'history' ? 'bg-accent text-content-inverse' : 'bg-surface-primary text-gray-700 hover:bg-surface-secondary'
              }`}
            >
              History
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
