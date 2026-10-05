import { getErrorMessage } from '@/api/errors'
import type { SearchStopReason } from '@/api/messages'
import { Button, Spinner } from '@/components/ui'
import type { MessageSearch } from './useMessageSearch'

const BUDGET: Record<Exclude<SearchStopReason, 'complete'>, string> = {
  scan_limit: 'One search reads up to 100,000 messages',
  time_limit: 'One search runs up to 20 seconds',
  match_limit: 'One search returns up to 500 matches',
}

const count = (n: number) => n.toLocaleString('en-US')

const ANNOUNCEMENT: Record<MessageSearch['status'], string> = {
  idle: '',
  running: 'Searching',
  done: 'Search finished',
  stopped: 'Search stopped',
  error: 'Search failed',
}

function percentDone(search: MessageSearch, direction: 'forward' | 'backward'): number {
  const range = search.range
  const progress = search.progress
  if (!range || !progress || range.last <= range.first) return 0
  const resume = progress.resume_seq
  if (resume == null) return 100
  const read = direction === 'backward' ? range.last - resume : resume - range.first
  return Math.min(100, Math.max(0, Math.round((read / (range.last - range.first)) * 100)))
}

interface Props {
  search: MessageSearch
  direction: 'forward' | 'backward'
}

export function SearchStatusBar({ search, direction }: Props) {
  const read = `${count(search.scanned)} messages read · ${count(search.messages.length)} found`
  const done = search.done

  const text = () => {
    switch (search.status) {
      case 'running':
        return `Searching… ${read}`
      case 'stopped':
        return `Stopped after ${read}`
      case 'error':
        return `Search failed: ${getErrorMessage(search.error)}`
      default:
        if (done && done.reason !== 'complete' && done.next_seq != null) {
          return `${read}. ${BUDGET[done.reason]}; the next one continues from #${count(done.next_seq)}.`
        }
        return search.range
          ? `Searched every message from #${count(search.range.first)} to #${count(search.range.last)}: ${read}`
          : `Nothing to search in this range: ${read}`
    }
  }

  const percent = percentDone(search, direction)

  return (
    <div
      className="px-4 py-2 bg-surface-secondary border-b text-xs text-content-secondary"
      data-testid="search-status"
      data-state={search.status}
      aria-busy={search.status === 'running'}
    >
      <span className="sr-only" role="status">
        {ANNOUNCEMENT[search.status]}
      </span>
      <div className="flex items-center gap-3">
        {search.status === 'running' && (
          <span aria-hidden="true" className="inline-flex">
            <Spinner size="sm" />
          </span>
        )}
        <span className={`flex-1 min-w-0 ${search.status === 'error' ? 'text-status-error-text' : ''}`}>{text()}</span>
        {search.status === 'running' && (
          <Button size="sm" variant="secondary" onClick={search.stop}>
            Stop
          </Button>
        )}
        {search.status !== 'running' && search.status !== 'error' && search.canContinue && (
          <Button size="sm" variant="secondary" onClick={search.more}>
            Search further
          </Button>
        )}
        {search.status === 'error' && (
          <Button size="sm" variant="secondary" onClick={search.canContinue ? search.more : search.restart}>
            Try again
          </Button>
        )}
      </div>
      {search.status === 'running' && (
        <div
          className="mt-1.5 h-1 rounded bg-surface-hover overflow-hidden"
          role="progressbar"
          aria-label="Search progress"
          aria-valuemin={0}
          aria-valuemax={100}
          aria-valuenow={percent}
        >
          <div className="h-full bg-accent transition-[width] motion-reduce:transition-none" style={{ width: `${percent}%` }} />
        </div>
      )}
    </div>
  )
}
