import Tooltip from '@/components/common/Tooltip'

export function NoAccessValue({ reason }: { reason: string }) {
  return (
    <Tooltip content={reason}>
      <span className="text-content-muted cursor-help">
        <span aria-hidden="true">—</span>
        <span className="sr-only">{reason}</span>
      </span>
    </Tooltip>
  )
}
