import { Button, Dropdown, PauseIcon, PlayIcon, SearchInput } from '@/components/ui'
import { formatCount } from '@/utils/formatters'
import { plural } from '@/utils/plural'
import { LIVE_MESSAGE_LIMITS, type LiveMessageLimit } from '../messages/unified/messageListUtils'

const RATE_OPTIONS = [
  { value: '0', label: 'No limit' },
  { value: '1', label: '1 msg/s' },
  { value: '5', label: '5 msg/s' },
  { value: '10', label: '10 msg/s' },
  { value: '25', label: '25 msg/s' },
  { value: '50', label: '50 msg/s' },
]

const MAX_SUBJECT_CHIPS = 12

interface Props {
  query: string
  onQueryChange: (query: string) => void
  subjectCounts: Record<string, number>
  subjectFilter: string | null
  onSubjectFilterChange: (subject: string | null) => void
  msgPerSecond?: number
  running: boolean
  isPaused: boolean
  onTogglePause: () => void
  onClear: () => void
  liveLimit: LiveMessageLimit
  onLiveLimitChange: (limit: LiveMessageLimit) => void
  maxDisplayRate: number
  onMaxDisplayRateChange: (rate: number) => void
}

export function SubscribeToolbar({
  query,
  onQueryChange,
  subjectCounts,
  subjectFilter,
  onSubjectFilterChange,
  msgPerSecond,
  running,
  isPaused,
  onTogglePause,
  onClear,
  liveLimit,
  onLiveLimitChange,
  maxDisplayRate,
  onMaxDisplayRateChange,
}: Props) {
  const counts = Object.entries(subjectCounts).sort((a, b) => b[1] - a[1])
  const total = counts.reduce((sum, [, n]) => sum + n, 0)

  return (
    <div className="border-b bg-surface-secondary">
      <div className="px-4 py-2 flex flex-wrap items-center gap-2">
        <div className="flex-1 min-w-[10rem] max-w-xs">
          <SearchInput value={query} onChange={onQueryChange} placeholder="Search received messages" size="sm" debounce={150} clearable />
        </div>
        <div className="ml-auto flex items-center gap-2">
          {running && (
            <Button
              size="sm"
              variant="secondary"
              icon={isPaused ? <PlayIcon className="w-3 h-3" /> : <PauseIcon className="w-3 h-3" />}
              onClick={onTogglePause}
            >
              {isPaused ? 'Resume' : 'Pause'}
            </Button>
          )}
          <Button size="sm" variant="ghost" onClick={onClear}>
            Clear
          </Button>
          <Dropdown
            size="sm"
            label="Keep last"
            value={String(liveLimit)}
            onChange={(v) => onLiveLimitChange(Number(v) as LiveMessageLimit)}
            options={LIVE_MESSAGE_LIMITS.map((l) => ({ value: String(l), label: `Keep ${l}` }))}
          />
          <Dropdown
            size="sm"
            label="Display rate"
            value={String(maxDisplayRate)}
            onChange={(v) => onMaxDisplayRateChange(Number(v))}
            options={RATE_OPTIONS}
          />
        </div>
      </div>
      <div
        className="px-4 pb-2 flex flex-wrap items-center gap-1.5"
        role={counts.length > 1 ? 'group' : undefined}
        aria-label={counts.length > 1 ? 'Filter by subject' : undefined}
      >
        <span className="mr-1 text-xs text-content-tertiary tabular-nums" data-testid="feed-counts">
          {plural(total, 'message')} received
          {running && !!msgPerSecond && ` · ${formatCount(msgPerSecond)} msg/s`}
        </span>
        {counts.length > 1 && (
          <>
            <button
              type="button"
              onClick={() => onSubjectFilterChange(null)}
              aria-pressed={subjectFilter === null}
              className={`rounded-full px-2 py-0.5 text-xs transition-colors ${
                subjectFilter === null ? 'bg-accent text-content-inverse' : 'bg-surface-tertiary text-content-secondary hover:bg-surface-hover'
              }`}
            >
              All
            </button>
            {counts.slice(0, MAX_SUBJECT_CHIPS).map(([subject, n]) => (
              <button
                key={subject}
                type="button"
                onClick={() => onSubjectFilterChange(subjectFilter === subject ? null : subject)}
                aria-pressed={subjectFilter === subject}
                className={`inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs transition-colors ${
                  subjectFilter === subject
                    ? 'bg-accent text-content-inverse'
                    : 'bg-surface-tertiary text-content-secondary hover:bg-surface-hover'
                }`}
              >
                <span className="font-mono">{subject}</span>
                <span className="tabular-nums opacity-80">{formatCount(n)}</span>
              </button>
            ))}
            {counts.length > MAX_SUBJECT_CHIPS && (
              <span className="text-xs text-content-tertiary">+{counts.length - MAX_SUBJECT_CHIPS} more</span>
            )}
          </>
        )}
      </div>
    </div>
  )
}
