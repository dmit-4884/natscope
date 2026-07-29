import { type ReactNode, type HTMLAttributes } from 'react'
import { cn } from '@/utils/cn'
import { CloseIcon, WarningIcon } from './icons'

export interface AlertProps extends HTMLAttributes<HTMLDivElement> {
  variant: 'info' | 'success' | 'warning' | 'error'
  title?: string
  /** Defaults to variant-specific icon */
  icon?: ReactNode
  dismissible?: boolean
  onDismiss?: () => void
  children: ReactNode
}

const variantStyles = {
  info: {
    container: 'bg-accent-light border-blue-200 text-blue-800',
    icon: 'text-blue-500',
    title: 'text-blue-800',
  },
  success: {
    container: 'bg-status-success-bg border-green-200 text-green-800',
    icon: 'text-green-500',
    title: 'text-green-800',
  },
  warning: {
    container: 'bg-status-warning-bg border-amber-200 text-amber-800',
    icon: 'text-amber-500',
    title: 'text-amber-800',
  },
  error: {
    container: 'bg-status-error-bg border-red-200 text-red-800',
    icon: 'text-red-500',
    title: 'text-red-800',
  },
}

const defaultIcons: Record<AlertProps['variant'], ReactNode> = {
  info: (
    <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
      />
    </svg>
  ),
  success: (
    <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
      />
    </svg>
  ),
  warning: <WarningIcon className="w-5 h-5" />,
  error: (
    <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth={2}
        d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
      />
    </svg>
  ),
}

/** Alert for displaying informational messages. */
export function Alert({
  variant,
  title,
  icon,
  dismissible = false,
  onDismiss,
  className,
  children,
  ...props
}: AlertProps) {
  const styles = variantStyles[variant]
  const displayIcon = icon ?? defaultIcons[variant]

  return (
    <div
      role="alert"
      className={cn(
        'p-4 border rounded-md flex gap-3',
        styles.container,
        className
      )}
      {...props}
    >
      {/* Icon */}
      {displayIcon && (
        <span className={cn('flex-shrink-0', styles.icon)}>{displayIcon}</span>
      )}

      {/* Content */}
      <div className="flex-1 min-w-0">
        {title && (
          <h4 className={cn('font-medium mb-1', styles.title)}>{title}</h4>
        )}
        <div className="text-sm">{children}</div>
      </div>

      {/* Dismiss button */}
      {dismissible && onDismiss && (
        <button
          onClick={onDismiss}
          className="flex-shrink-0 text-current opacity-60 hover:opacity-100 transition-opacity"
          aria-label="Dismiss"
        >
          <CloseIcon className="w-4 h-4" />
        </button>
      )}
    </div>
  )
}

Alert.displayName = 'Alert'
