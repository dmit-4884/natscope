import Tooltip from '@/components/common/Tooltip'

export function NoAccessValue({ reason }: { reason: string }) {
  return (
    <Tooltip content={reason}>
      <span className="text-content-muted cursor-help" aria-label={reason}>
        —
      </span>
    </Tooltip>
  )
}
