import { cn } from '@/utils/cn'
import Tooltip from '@/components/common/Tooltip'
import { RefreshIcon } from './icons'

/**
 * Uniform icon button for row-level actions in tables and list cards
 * (edit / duplicate / delete / ping). One size, one icon set, one hover
 * treatment everywhere — so rows never mix text links with ad-hoc icons.
 */

type RowActionKind = 'edit' | 'duplicate' | 'delete' | 'ping'

const ICONS: Record<Exclude<RowActionKind, 'ping'>, React.ReactNode> = {
  edit: (
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={1.5}
      d="M16.862 4.487l1.687-1.688a1.875 1.875 0 112.652 2.652L10.582 16.07a4.5 4.5 0 01-1.897 1.13L6 18l.8-2.685a4.5 4.5 0 011.13-1.897l8.932-8.931zm0 0L19.5 7.125M18 14v4.75A2.25 2.25 0 0115.75 21H5.25A2.25 2.25 0 013 18.75V8.25A2.25 2.25 0 015.25 6H10"
    />
  ),
  duplicate: (
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={1.5}
      d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"
    />
  ),
  delete: (
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={1.5}
      d="M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0"
    />
  ),
}

const HOVER: Record<RowActionKind, string> = {
  edit: 'hover:text-accent hover:bg-accent-light',
  duplicate: 'hover:text-accent hover:bg-accent-light',
  delete: 'hover:text-status-error-text hover:bg-status-error-bg',
  ping: 'hover:text-status-success-text hover:bg-status-success-bg',
}

const DEFAULT_TITLES: Record<RowActionKind, string> = {
  edit: 'Edit',
  duplicate: 'Duplicate',
  delete: 'Delete',
  ping: 'Test connection',
}

export interface RowActionButtonProps {
  kind: RowActionKind
  onClick: () => void
  /** Accessible name, e.g. `Edit mapping orders.*`. Falls back to the default title. */
  label?: string
  /** Tooltip. Falls back to the default per-kind title. */
  title?: string
  disabled?: boolean
  /** Spin the icon (ping in progress). */
  spinning?: boolean
}

export function RowActionButton({
  kind,
  onClick,
  label,
  title,
  disabled = false,
  spinning = false,
}: RowActionButtonProps) {
  return (
    <Tooltip content={title ?? DEFAULT_TITLES[kind]}>
      <button
        type="button"
        onClick={onClick}
        disabled={disabled}
        aria-label={label ?? DEFAULT_TITLES[kind]}
        className={cn(
          'p-1.5 rounded-md text-content-muted transition-colors disabled:opacity-50 disabled:pointer-events-none',
          HOVER[kind],
        )}
      >
        {kind === 'ping' ? (
          <RefreshIcon className={cn('w-4 h-4', spinning && 'animate-spin')} />
        ) : (
          <svg
            className={cn('w-4 h-4', spinning && 'animate-spin')}
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            {ICONS[kind]}
          </svg>
        )}
      </button>
    </Tooltip>
  )
}

RowActionButton.displayName = 'RowActionButton'
