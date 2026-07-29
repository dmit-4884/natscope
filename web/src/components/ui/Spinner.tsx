import { cn } from '@/utils/cn'

export interface SpinnerProps {
  /** Spinner size */
  size?: 'sm' | 'md' | 'lg'
  /** Spinner color (defaults to currentColor) */
  color?: string
  /** Additional classes */
  className?: string
}

const sizeStyles = {
  sm: 'w-4 h-4',
  md: 'w-5 h-5',
  lg: 'w-6 h-6',
}

/**
 * Loading spinner component
 *
 * @example
 * <Spinner />
 *
 * @example
 */
export function Spinner({ size = 'md', color, className }: SpinnerProps) {
  return (
    <svg
      className={cn('animate-spin', sizeStyles[size], className)}
      style={color ? { color } : undefined}
      fill="none"
      viewBox="0 0 24 24"
      role="status"
      aria-label="Loading"
    >
      <circle
        className="opacity-25"
        cx="12"
        cy="12"
        r="10"
        stroke="currentColor"
        strokeWidth="4"
      />
      <path
        className="opacity-75"
        fill="currentColor"
        d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
      />
    </svg>
  )
}

Spinner.displayName = 'Spinner'
