import { type ReactNode, type HTMLAttributes } from 'react'
import { cn } from '@/utils/cn'

export interface BadgeProps extends HTMLAttributes<HTMLSpanElement> {
  variant?: 'default' | 'primary' | 'success' | 'warning' | 'error'
  shape?: 'rounded' | 'pill'
  size?: 'sm' | 'md'
  children: ReactNode
}

const variantStyles = {
  default: 'bg-surface-tertiary text-gray-700',
  primary: 'bg-accent-muted text-accent-text',
  success: 'bg-status-success-light text-green-700',
  warning: 'bg-status-warning-light text-amber-700',
  error: 'bg-status-error-light text-red-700',
}

const shapeStyles = {
  rounded: 'rounded',
  pill: 'rounded-full',
}

const sizeStyles = {
  sm: 'px-1.5 py-0.5 text-xs',
  md: 'px-2 py-0.5 text-xs',
}

/**
 * Badge component for displaying status, counts, or labels
 *
 * @example
 * // Default badge
 * <Badge>Label</Badge>
 *
 * @example
 * // Success pill badge
 * <Badge variant="success" shape="pill">Active</Badge>
 *
 * @example
 * // Count badge
 * <Badge variant="primary" shape="pill">{count}</Badge>
 */
export function Badge({
  variant = 'default',
  shape = 'rounded',
  size = 'md',
  className,
  children,
  ...props
}: BadgeProps) {
  return (
    <span
      className={cn(
        'inline-flex items-center font-medium',
        variantStyles[variant],
        shapeStyles[shape],
        sizeStyles[size],
        className
      )}
      {...props}
    >
      {children}
    </span>
  )
}

Badge.displayName = 'Badge'
