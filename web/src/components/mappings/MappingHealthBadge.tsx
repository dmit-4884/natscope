import type { ReactNode } from 'react'
import type { MappingHealth, MappingHealthInfo } from '@/contexts/mappings'
import { CheckIcon, CloseIcon, PauseIcon, WarningIcon } from '@/components/ui'

interface Props {
  health?: MappingHealth
  detail?: string
  /** Pass the full info object if you have it; otherwise health/detail props are used. */
  info?: MappingHealthInfo
  className?: string
}

const COLOR_BY_HEALTH: Record<MappingHealth, string> = {
  ok: 'bg-status-success-bg text-green-700 border-green-200',
  source_missing: 'bg-status-error-bg text-red-700 border-red-200',
  source_disabled: 'bg-status-warning-bg text-amber-700 border-amber-200',
  selection_missing: 'bg-status-warning-bg text-amber-700 border-amber-200',
  descriptor_missing: 'bg-status-error-bg text-red-700 border-red-200',
  type_missing: 'bg-status-error-bg text-red-700 border-red-200',
}

const SHORT_LABEL: Record<MappingHealth, string> = {
  ok: 'Ready',
  source_missing: 'Source missing',
  source_disabled: 'Source off',
  selection_missing: 'No version',
  descriptor_missing: 'Not compiled',
  type_missing: 'Type missing',
}

const DEFAULT_TOOLTIP: Record<MappingHealth, string> = {
  ok: 'Source enabled, descriptor compiled, type resolves — decode is ready.',
  source_missing: 'The proto source this mapping points at no longer exists.',
  source_disabled: 'The source is disabled. Toggle it on in Proto Files.',
  selection_missing: 'Git source has no selected version. Pick a tag in Proto Files.',
  descriptor_missing: 'No compiled descriptor for the active version. Run Compile in Proto Files.',
  type_missing: 'The message type is not present in the compiled descriptor.',
}

const ICON_BY_HEALTH: Record<MappingHealth, ReactNode> = {
  ok: <CheckIcon className="w-3 h-3" />,
  source_missing: <CloseIcon className="w-3 h-3" />,
  source_disabled: <PauseIcon className="w-3 h-3" />,
  selection_missing: <WarningIcon className="w-3 h-3" />,
  descriptor_missing: <CloseIcon className="w-3 h-3" />,
  type_missing: <CloseIcon className="w-3 h-3" />,
}

/**
 * Renders a small badge indicating mapping resolvability.
 * Tooltip carries either the backend-provided detail or a default explanation.
 */
export function MappingHealthBadge({ health, detail, info, className }: Props) {
  const h: MappingHealth | undefined = health ?? info?.health
  const d = detail ?? info?.detail

  if (!h) {
    return (
      <span
        className={
          'inline-flex items-center gap-1 px-2 py-0.5 text-xs whitespace-nowrap rounded border bg-surface-secondary text-content-tertiary border-border ' +
          (className ?? '')
        }
        title="Health is being computed…"
      >
        <span aria-hidden>?</span>
        <span>Pending</span>
      </span>
    )
  }

  return (
    <span
      title={d || DEFAULT_TOOLTIP[h]}
      className={
        `inline-flex items-center gap-1 px-2 py-0.5 text-xs whitespace-nowrap rounded border ${COLOR_BY_HEALTH[h]} ` +
        (className ?? '')
      }
    >
      <span aria-hidden>{ICON_BY_HEALTH[h]}</span>
      <span>{SHORT_LABEL[h]}</span>
    </span>
  )
}
