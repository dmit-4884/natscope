import { useState } from 'react'
import Tooltip from '@/components/common/Tooltip'
import { Button, CloseIcon, Dropdown, EyeOffIcon, PauseIcon, PlayIcon, SearchInput } from '@/components/ui'
import { matchSubject } from '@/shared/domain/subjectMatch'
import { formatCount, formatNumber } from '@/utils/formatters'
import { plural } from '@/utils/plural'
import { LIVE_MESSAGE_LIMITS, type LiveMessageLimit } from '../messages/unified/messageListUtils'

const RATE_OPTIONS = [
  { value: '0', label: 'All messages' },
  { value: '1', label: 'At most 1 msg/s' },
  { value: '5', label: 'At most 5 msg/s' },
  { value: '10', label: 'At most 10 msg/s' },
  { value: '25', label: 'At most 25 msg/s' },
  { value: '50', label: 'At most 50 msg/s' },
]

function rateOptions(current: number) {
  if (RATE_OPTIONS.some((o) => Number(o.value) === current)) return RATE_OPTIONS
  return [...RATE_OPTIONS, { value: String(current), label: `At most ${formatCount(current)} msg/s` }].sort(
    (a, b) => Number(a.value) - Number(b.value),
  )
}

const MAX_SUBJECT_CHIPS = 12

const chipClass = (active: boolean) =>
  `rounded-full px-2 py-0.5 text-xs transition-colors ${
    active ? 'bg-accent text-content-inverse' : 'bg-surface-tertiary text-content-secondary hover:bg-surface-hover'
  }`

interface FeedCounts {
  received: number
  shown: number
  matching: number | null
  skipped: number | undefined
  msgPerSecond: number | undefined
}

interface Props {
  query: string
  onQueryChange: (query: string) => void
  subjectCounts: Record<string, number>
  subjectFilter: string | null
  onSubjectFilterChange: (subject: string | null) => void
  muted: string[]
  onMute: (subject: string) => void
  onUnmute: (subject: string) => void
  counts: FeedCounts
  running: boolean
  isPaused: boolean
  pausedCount: number
  onTogglePause: () => void
  onClear: () => void
  liveLimit: LiveMessageLimit
  onLiveLimitChange: (limit: LiveMessageLimit) => void
  displayRate: number
  onDisplayRateChange: (rate: number) => void
}

function FeedSummary({ counts, liveLimit }: { counts: FeedCounts; liveLimit: number }) {
  return (
    <span className="mr-1 text-xs text-content-tertiary tabular-nums" data-testid="feed-counts">
      {plural(counts.received, 'message', undefined, formatCount)} received
      {!!counts.msgPerSecond && ` · ${formatCount(counts.msgPerSecond)} msg/s`}
      {counts.received > counts.shown && ` · showing the last ${formatCount(liveLimit)}`}
      {counts.matching !== null && ` · ${formatCount(counts.matching)} match`}
      {!!counts.skipped && (
        <Tooltip content="Skipped by the display rate limit or because the browser could not keep up. Core NATS keeps no copy.">
          <span className="text-status-warning-text" data-testid="feed-skipped">
            {` · ${formatCount(counts.skipped)} skipped`}
          </span>
        </Tooltip>
      )}
    </span>
  )
}

export function SubscribeToolbar({
  query,
  onQueryChange,
  subjectCounts,
  subjectFilter,
  onSubjectFilterChange,
  muted,
  onMute,
  onUnmute,
  counts,
  running,
  isPaused,
  pausedCount,
  onTogglePause,
  onClear,
  liveLimit,
  onLiveLimitChange,
  displayRate,
  onDisplayRateChange,
}: Props) {
  const [showAll, setShowAll] = useState(false)
  const subjects = Object.entries(subjectCounts).filter(([subject]) => !muted.some((pattern) => matchSubject(subject, pattern)))
  const visible = showAll ? subjects : subjects.slice(0, MAX_SUBJECT_CHIPS)
  const hidden = subjects.length - visible.length

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
              {isPaused ? `Resume${pausedCount > 0 ? ` (+${formatNumber(pausedCount)})` : ''}` : 'Pause'}
            </Button>
          )}
          <Button size="sm" variant="ghost" onClick={onClear}>
            Clear
          </Button>
          <Tooltip content="How many of the latest messages the feed keeps">
            <span>
              <Dropdown
                size="sm"
                label="Keep last"
                value={String(liveLimit)}
                onChange={(v) => onLiveLimitChange(Number(v) as LiveMessageLimit)}
                options={LIVE_MESSAGE_LIMITS.map((l) => ({ value: String(l), label: `Keep ${formatCount(l)}` }))}
              />
            </span>
          </Tooltip>
          <Tooltip content="On a busy subject, show at most this many messages per second; the rest are skipped. Applies to this subscription only.">
            <span>
              <Dropdown
                size="sm"
                label="Display rate"
                value={String(displayRate)}
                onChange={(v) => onDisplayRateChange(Number(v))}
                options={rateOptions(displayRate)}
              />
            </span>
          </Tooltip>
        </div>
      </div>
      <div
        className="px-4 pb-2 flex flex-wrap items-center gap-1.5"
        role={subjects.length > 1 ? 'group' : undefined}
        aria-label={subjects.length > 1 ? 'Filter by subject' : undefined}
      >
        <FeedSummary counts={counts} liveLimit={liveLimit} />
        {subjects.length > 1 && (
          <>
            <button type="button" onClick={() => onSubjectFilterChange(null)} aria-pressed={subjectFilter === null} className={chipClass(subjectFilter === null)}>
              All
            </button>
            {visible.map(([subject, n]) => (
              <span key={subject} className="inline-flex items-center">
                <button
                  type="button"
                  onClick={() => onSubjectFilterChange(subjectFilter === subject ? null : subject)}
                  aria-pressed={subjectFilter === subject}
                  className={`${chipClass(subjectFilter === subject)} inline-flex items-center gap-1.5 rounded-r-none pr-1.5`}
                >
                  <span className="font-mono">{subject}</span>
                  <span className="tabular-nums opacity-80">{formatCount(n)}</span>
                </button>
                <Tooltip content={`Hide ${subject} from the feed`}>
                  <button
                    type="button"
                    onClick={() => onMute(subject)}
                    aria-label={`Mute ${subject}`}
                    className={`${chipClass(subjectFilter === subject)} rounded-l-none pl-1 pr-1.5 py-1 opacity-60 hover:opacity-100 focus-visible:opacity-100`}
                  >
                    <EyeOffIcon className="w-3 h-3" />
                  </button>
                </Tooltip>
              </span>
            ))}
            {hidden > 0 && (
              <button type="button" onClick={() => setShowAll(true)} className="text-xs text-accent hover:text-accent-text">
                +{hidden} more
              </button>
            )}
            {showAll && subjects.length > MAX_SUBJECT_CHIPS && (
              <button type="button" onClick={() => setShowAll(false)} className="text-xs text-accent hover:text-accent-text">
                Show fewer
              </button>
            )}
          </>
        )}
      </div>
      {muted.length > 0 && (
        <div className="px-4 pb-2 flex flex-wrap items-center gap-1.5" data-testid="muted-subjects">
          <span className="text-xs text-content-tertiary">Muted:</span>
          {muted.map((subject) => (
            <span key={subject} className="inline-flex items-center gap-1 rounded-full border border-border px-2 py-0.5 text-xs text-content-secondary">
              <span className="font-mono">{subject}</span>
              <button
                type="button"
                onClick={() => onUnmute(subject)}
                aria-label={`Unmute ${subject}`}
                className="p-0.5 rounded hover:bg-surface-hover text-content-muted hover:text-content-secondary"
              >
                <CloseIcon className="w-3 h-3" />
              </button>
            </span>
          ))}
        </div>
      )}
    </div>
  )
}
