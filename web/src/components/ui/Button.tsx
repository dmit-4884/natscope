import { forwardRef, type ReactNode, type ButtonHTMLAttributes } from 'react'
import { cn } from '@/utils/cn'
import { buttonClassName, type ButtonSize, type ButtonVariant } from './buttonStyles'

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  /** Button style variant */
  variant?: ButtonVariant
  /** Button size */
  size?: ButtonSize
  /** Show loading spinner and disable button */
  loading?: boolean
  /** Icon to show on the left side */
  icon?: ReactNode
  /** Icon to show on the right side */
  iconRight?: ReactNode
  /** Make button full width */
  fullWidth?: boolean
  /** Button content */
  children?: ReactNode
}

const iconSizeStyles = {
  sm: 'w-3.5 h-3.5',
  md: 'w-4 h-4',
  lg: 'w-5 h-5',
}

/**
 * Primary UI button component
 *
 * @example
 * // Primary button
 * <Button variant="primary">Save</Button>
 *
 * @example
 * // With icon and loading
 * <Button icon={<SaveIcon />} loading={isSaving}>
 *   Save Changes
 * </Button>
 *
 * @example
 * // Danger button
 * <Button variant="danger" size="sm">Delete</Button>
 */
export const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  (
    {
      variant = 'primary',
      size = 'md',
      loading = false,
      disabled = false,
      icon,
      iconRight,
      fullWidth = false,
      className,
      children,
      type = 'button',
      ...props
    },
    ref
  ) => {
    const isDisabled = disabled || loading

    return (
      <button
        ref={ref}
        type={type}
        disabled={isDisabled}
        className={cn(
          buttonClassName(variant, size),
          // Full width
          fullWidth && 'w-full',
          // Loading state
          loading && 'relative text-transparent',
          className
        )}
        {...props}
      >
        {/* Loading spinner overlay */}
        {loading && (
          <span className="absolute inset-0 flex items-center justify-center">
            <svg
              className={cn('animate-spin', iconSizeStyles[size], 'text-current')}
              style={{ color: variant === 'primary' || variant === 'danger' ? 'white' : 'currentColor' }}
              fill="none"
              viewBox="0 0 24 24"
              aria-hidden="true"
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
          </span>
        )}

        {/* Left icon */}
        {icon && <span className={cn(iconSizeStyles[size], 'flex-shrink-0')}>{icon}</span>}

        {/* Children */}
        {children}

        {/* Right icon */}
        {iconRight && <span className={cn(iconSizeStyles[size], 'flex-shrink-0')}>{iconRight}</span>}
      </button>
    )
  }
)

Button.displayName = 'Button'
