import type { ReactNode } from 'react'
import { cn } from '@/utils/cn'
import { ChevronDownIcon } from '@/components/ui/icons'

/**
 * Unified section card used across all Settings tabs: white card, gray
 * header with title/description, optional icon, count badge, header
 * actions, and an optional collapsible mode.
 */

export interface SettingsSectionProps {
  title: string
  description?: string
  /** Small leading icon (16×16) rendered muted next to the title. */
  icon?: ReactNode
  /** Count or short label rendered as a small gray badge after the title. */
  badge?: ReactNode
  /** Buttons rendered on the right side of the header. */
  actions?: ReactNode
  collapsible?: boolean
  isOpen?: boolean
  onToggle?: () => void
  /** Remove body padding for flush content (tables, lists). */
  flush?: boolean
  children: ReactNode
}

function Chevron({ open }: { open: boolean }) {
  return <ChevronDownIcon className={cn('w-4 h-4 text-content-muted shrink-0 transition-transform', open && 'rotate-180')} />
}

export function SettingsSection({
  title,
  description,
  icon,
  badge,
  actions,
  collapsible = false,
  isOpen = true,
  onToggle,
  flush = false,
  children,
}: SettingsSectionProps) {
  const showBody = !collapsible || isOpen

  const heading = (
    <div className="min-w-0 flex-1">
      <div className="flex items-center gap-2">
        {icon && <span className="text-content-muted shrink-0 [&>svg]:w-4 [&>svg]:h-4">{icon}</span>}
        <h3 className="text-sm font-semibold text-gray-800 truncate">{title}</h3>
        {badge != null && badge !== '' && (
          <span className="text-xs font-medium tabular-nums text-content-tertiary bg-surface-tertiary px-1.5 py-0.5 rounded shrink-0">
            {badge}
          </span>
        )}
      </div>
      {description && <p className="text-xs text-content-tertiary mt-0.5">{description}</p>}
    </div>
  )

  return (
    <section className="bg-surface-primary border border-border rounded-lg overflow-hidden">
      <div
        className={cn(
          'flex items-center justify-between gap-3 px-4 py-3 bg-surface-secondary',
          showBody && 'border-b border-border',
        )}
      >
        {collapsible ? (
          <button
            type="button"
            onClick={onToggle}
            aria-expanded={isOpen}
            className="flex items-center justify-between gap-3 flex-1 min-w-0 text-left"
          >
            {heading}
            <Chevron open={isOpen} />
          </button>
        ) : (
          heading
        )}
        {actions && <div className="flex items-center gap-2 shrink-0">{actions}</div>}
      </div>
      {showBody && <div className={flush ? undefined : 'p-4 space-y-4'}>{children}</div>}
    </section>
  )
}
