import type { ReactNode } from 'react'
import { cn } from '@/utils/cn'

/**
 * Shared scaffold for every Settings tab: one header, one content width,
 * one set of paddings — so tabs stay visually identical.
 *
 * `scroll="page"` scrolls the whole content area (cards flow naturally);
 * `scroll="fill"` gives children the full remaining height so they manage
 * their own scrolling (tables with sticky toolbars).
 */

const CONTAINER = 'mx-auto w-full max-w-6xl px-4 sm:px-6 lg:px-8'

interface SettingsPageProps {
  title: string
  description?: string
  meta?: string
  actions?: ReactNode
  scroll?: 'page' | 'fill'
  children: ReactNode
}

export function SettingsPage({
  title,
  description,
  meta,
  actions,
  scroll = 'page',
  children,
}: SettingsPageProps) {
  return (
    <div className="flex-1 flex flex-col overflow-hidden">
      <header className="shrink-0 bg-surface-primary border-b border-border">
        <div
          className={cn(
            CONTAINER,
            'py-4 flex flex-wrap items-start justify-between gap-x-6 gap-y-3',
          )}
        >
          <div className="min-w-0">
            <h1 className="text-lg sm:text-xl font-semibold text-content-primary tracking-tight truncate">
              {title}
            </h1>
            {description && (
              <p className="mt-0.5 text-sm text-content-tertiary truncate" title={description}>
                {description}
              </p>
            )}
            {meta && (
              <p className="mt-1.5 text-xs text-content-muted font-medium uppercase tracking-wider">
                {meta}
              </p>
            )}
          </div>
          {actions && <div className="flex flex-wrap items-center gap-2">{actions}</div>}
        </div>
      </header>

      {scroll === 'page' ? (
        <div className="flex-1 overflow-y-auto">
          <div className={cn(CONTAINER, 'py-6 space-y-6')}>{children}</div>
        </div>
      ) : (
        <div className="flex-1 overflow-hidden">
          <div className={cn(CONTAINER, 'py-6 h-full flex flex-col min-h-0')}>{children}</div>
        </div>
      )}
    </div>
  )
}
