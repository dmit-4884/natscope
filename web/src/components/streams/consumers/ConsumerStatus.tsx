import { Badge } from '@/components/ui'
import Tooltip from '@/components/common/Tooltip'
import { statusText, type ConsumerIssue, type ConsumerState } from './consumerHealth'

const CALM: Record<'working' | 'caught_up', { hint: string; variant: 'primary' | 'success' }> = {
  working: { hint: 'Messages are on their way or waiting for an ack, and nothing looks wrong.', variant: 'primary' },
  caught_up: { hint: 'Nothing left to deliver and nothing waiting for an ack.', variant: 'success' },
}

interface Props {
  issues: ConsumerIssue[]
  state: ConsumerState
  limit?: number
}

export function ConsumerStatus({ issues, state, limit }: Props) {
  if (issues.length === 0) {
    const calm = CALM[state === 'working' ? 'working' : 'caught_up']
    return (
      <Tooltip content={calm.hint}>
        <Badge variant={calm.variant} size="sm" data-testid="consumer-status">
          {statusText([], state)}
        </Badge>
      </Tooltip>
    )
  }
  const shown = limit ? issues.slice(0, limit) : issues
  const hidden = issues.length - shown.length
  return (
    <span className="inline-flex flex-wrap items-center gap-1" data-testid="consumer-status">
      {shown.map((issue) => (
        <Tooltip key={issue.kind} content={issue.detail}>
          <Badge variant={issue.severity === 'error' ? 'error' : 'warning'} size="sm" data-issue={issue.kind}>
            {issue.label}
            <span className="sr-only">: {issue.detail}</span>
          </Badge>
        </Tooltip>
      ))}
      {hidden > 0 && <span className="text-xs text-content-tertiary">+{hidden}</span>}
    </span>
  )
}
