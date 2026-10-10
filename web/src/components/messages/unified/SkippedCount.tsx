import Tooltip from '@/components/common/Tooltip'
import { formatCount } from '@/utils/formatters'

interface Props {
  count: number | undefined
  note: string
}

export function SkippedCount({ count, note }: Props) {
  if (!count) return null
  return (
    <Tooltip content={`Skipped by the display rate limit, or because the server could not deliver messages fast enough. ${note}`}>
      <span className="text-status-warning-text" data-testid="feed-skipped">
        {`${formatCount(count)} skipped`}
      </span>
    </Tooltip>
  )
}
