import { type ReactNode } from 'react'
import { cn } from '@/utils/cn'
import { CloseIcon } from './icons'

export interface FilterChipProps {
  /** Filter label */
  label: string
  /** Filter value */
  value?: string
  /** Remove callback */
  onRemove: () => void
  /** Chip variant */
  variant?: 'default' | 'primary'
  /** Additional classes */
  className?: string
}

/**
 * Filter chip component for displaying active filters
 *
 * @example
 */
export function FilterChip({
  label,
  value,
  onRemove,
  variant = 'default',
  className,
}: FilterChipProps) {
  return (
    <span
      className={cn(
        'inline-flex items-center gap-1 pl-2 pr-1 py-0.5 rounded-full text-xs font-medium',
        variant === 'default' && 'bg-surface-tertiary text-gray-700',
        variant === 'primary' && 'bg-accent-light border border-blue-200 text-accent-text',
        className
      )}
    >
      <span className={variant === 'primary' ? 'text-blue-500' : 'text-content-tertiary'}>{label}:</span>
      {value && <span className="font-mono truncate max-w-[200px]" title={value}>{value}</span>}
      <button
        type="button"
        onClick={onRemove}
        className={cn(
          'ml-0.5 p-0.5 rounded-full transition-colors',
          variant === 'default' && 'hover:bg-surface-hover text-content-muted hover:text-content-secondary',
          variant === 'primary' && 'hover:bg-blue-200 text-blue-400 hover:text-accent'
        )}
        aria-label={`Remove ${label} filter`}
      >
        <CloseIcon className="w-3 h-3" />
      </button>
    </span>
  )
}

FilterChip.displayName = 'FilterChip'

/**
 * Filter chips container with "Clear all" button
 */
export interface FilterChipsGroupProps {
  children: ReactNode
  onClearAll?: () => void
  className?: string
}

export function FilterChipsGroup({ children, onClearAll, className }: FilterChipsGroupProps) {
  return (
    <div className={cn('flex flex-wrap items-center gap-2', className)}>
      {children}
      {onClearAll && (
        <button
          type="button"
          onClick={onClearAll}
          className="text-xs text-content-tertiary hover:text-gray-700 hover:underline"
        >
          Clear all
        </button>
      )}
    </div>
  )
}

FilterChipsGroup.displayName = 'FilterChipsGroup'
