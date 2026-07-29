import { type ReactNode } from 'react'
import { cn } from '@/utils/cn'

export interface EmptyStateProps {
  /** Icon to display */
  icon?: ReactNode
  /** Main title */
  title: string
  /** Description text */
  description?: string
  /** Action button or link */
  action?: ReactNode
  /** Secondary action text with onClick - displays as a link */
  secondaryAction?: {
    label: string
    onClick: () => void
  }
  /** Additional classes */
  className?: string
  /** Size variant */
  size?: 'sm' | 'md' | 'lg'
}

const sizeStyles = {
  sm: {
    container: 'py-6 px-3',
    icon: 'w-8 h-8 mb-2',
    title: 'text-xs font-medium',
    description: 'text-xs max-w-xs',
  },
  md: {
    container: 'py-12 px-4',
    icon: 'w-12 h-12 mb-4',
    title: 'text-sm font-medium',
    description: 'text-sm max-w-sm',
  },
  lg: {
    container: 'py-16 px-6',
    icon: 'w-16 h-16 mb-6',
    title: 'text-base font-semibold',
    description: 'text-sm max-w-md',
  },
}

/**
 * Empty state component for when there's no data to display
 *
 * @example
 * <EmptyState
 *   icon={<InboxIcon />}
 *   title="No messages"
 *   description="Messages will appear here when they arrive."
 * />
 *
 * @example
 * <EmptyState
 *   icon={<SearchIcon />}
 *   title="No results found"
 *   description="Try adjusting your search or filter."
 *   action={<Button variant="secondary" onClick={clearFilters}>Clear filters</Button>}
 * />
 *
 * @example
 * <EmptyState
 *   icon={<PlusIcon />}
 *   title="No streams"
 *   description="Create your first stream to get started."
 *   secondaryAction={{ label: "Learn more about streams", onClick: openDocs }}
 * />
 */
export function EmptyState({
  icon,
  title,
  description,
  action,
  secondaryAction,
  className,
  size = 'md',
}: EmptyStateProps) {
  const styles = sizeStyles[size]

  return (
    <div
      className={cn(
        'flex flex-col items-center justify-center text-center',
        styles.container,
        className
      )}
      role="status"
      aria-label={title}
    >
      {icon && (
        <div className={cn('text-content-muted', styles.icon)}>{icon}</div>
      )}
      <h3 className={cn('text-content-primary mb-1', styles.title)}>{title}</h3>
      {description && (
        <p className={cn('text-content-tertiary mb-4', styles.description)}>{description}</p>
      )}
      {action && <div className="mt-2">{action}</div>}
      {secondaryAction && (
        <button
          onClick={secondaryAction.onClick}
          className="mt-3 text-sm text-accent hover:text-accent-text hover:underline focus:outline-none focus:ring-2 focus:ring-border-focus focus:ring-offset-2 rounded"
        >
          {secondaryAction.label} &rarr;
        </button>
      )}
    </div>
  )
}

EmptyState.displayName = 'EmptyState'
