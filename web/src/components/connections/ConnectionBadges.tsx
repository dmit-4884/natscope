import { LockClosedIcon } from '@/components/ui'
import Tooltip from '@/components/common/Tooltip'
import type { ConnectionPolicy } from '@/contexts/connection'
import { LABEL_BADGE_CLASSES } from './labelStyles'

interface Props {
  policy: ConnectionPolicy
}

export function ConnectionBadges({ policy }: Props) {
  const { label, readOnly } = policy
  if (!label && !readOnly) return null
  return (
    <>
      {label && (
        <span
          data-color={label.color}
          className={`shrink-0 rounded px-1.5 py-0.5 text-2xs font-bold uppercase tracking-wide ${LABEL_BADGE_CLASSES[label.color]}`}
        >
          {label.text}
        </span>
      )}
      {readOnly && (
        <Tooltip content="Natscope refuses every write through this connection">
          <span className="shrink-0 inline-flex items-center gap-1 rounded border border-border-strong bg-surface-tertiary px-1.5 py-0.5 text-2xs font-medium text-content-secondary">
            <LockClosedIcon className="w-3 h-3" />
            Read-only
          </span>
        </Tooltip>
      )}
    </>
  )
}
