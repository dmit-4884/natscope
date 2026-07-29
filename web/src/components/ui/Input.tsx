import { forwardRef, useId, type ReactNode, type InputHTMLAttributes } from 'react'
import { cn } from '@/utils/cn'
import { CloseIcon } from './icons'

export interface InputProps extends Omit<InputHTMLAttributes<HTMLInputElement>, 'size'> {
  /** Show error state styling */
  error?: boolean
  /** Error message to display below input */
  errorMessage?: string
  /** Icon to show on the left side */
  icon?: ReactNode
  /** Icon or button to show on the right side */
  iconRight?: ReactNode
  /** Show clear button (X) when value exists */
  onClear?: () => void
  /** Input size */
  size?: 'sm' | 'md' | 'lg'
  /** Use monospace font (for URLs, code, etc.) */
  mono?: boolean
}

const sizeStyles = {
  sm: 'px-2.5 py-1.5 text-xs',
  md: 'px-3 py-2 text-sm',
  lg: 'px-4 py-3 text-base',
}

const iconPaddingLeft = {
  sm: 'pl-8',
  md: 'pl-10',
  lg: 'pl-12',
}

const iconPaddingRight = {
  sm: 'pr-8',
  md: 'pr-10',
  lg: 'pr-12',
}

const iconPositionStyles = {
  sm: 'w-4 h-4',
  md: 'w-4 h-4',
  lg: 'w-5 h-5',
}

/**
 * Input component with support for icons, error states, and clear button
 */
export const Input = forwardRef<HTMLInputElement, InputProps>(
  (
    {
      error = false,
      errorMessage,
      icon,
      iconRight,
      onClear,
      size = 'md',
      mono = false,
      className,
      disabled,
      value,
      ...props
    },
    ref
  ) => {
    const hasLeftIcon = !!icon
    const hasRightContent = !!iconRight || (onClear && value)
    // Stable id lets the input reference the error via aria-describedby, so
    // screen readers announce it on focus, not only when tabbing past.
    const errorId = useId()
    const hasError = error || !!errorMessage

    return (
      <div className="relative">
        {hasLeftIcon && (
          <span
            className={cn(
              'absolute left-3 top-1/2 -translate-y-1/2 text-content-muted pointer-events-none',
              iconPositionStyles[size]
            )}
          >
            {icon}
          </span>
        )}

        <input
          ref={ref}
          disabled={disabled}
          value={value}
          aria-invalid={hasError || undefined}
          aria-describedby={errorMessage ? errorId : undefined}
          className={cn(
            // Base styles
            'w-full rounded-md border bg-surface-primary text-content-primary',
            'placeholder:text-content-tertiary',
            'transition-colors duration-150',
            // Focus styles
            'focus:outline-none focus:ring-2 focus:ring-border-focus focus:border-border-focus',
            // Disabled styles
            'disabled:bg-surface-tertiary disabled:text-content-tertiary disabled:cursor-not-allowed',
            // Error styles
            hasError
              ? 'border-status-error-border focus:ring-status-error-border focus:border-status-error-border'
              : 'border-border-strong',
            // Size styles
            sizeStyles[size],
            // Icon padding
            hasLeftIcon && iconPaddingLeft[size],
            hasRightContent && iconPaddingRight[size],
            // Mono font
            mono && 'font-mono',
            className
          )}
          {...props}
        />

        {hasRightContent && (
          <span
            className={cn(
              'absolute right-3 top-1/2 -translate-y-1/2 flex items-center gap-1',
              iconPositionStyles[size]
            )}
          >
            {onClear && value && (
              <button
                type="button"
                onClick={onClear}
                className="text-content-muted hover:text-content-secondary transition-colors"
                tabIndex={-1}
                aria-label="Clear input"
              >
                <CloseIcon className="w-4 h-4" />
              </button>
            )}
            {iconRight && !onClear && (
              <span className="text-content-muted">{iconRight}</span>
            )}
          </span>
        )}

        {errorMessage && (
          <p id={errorId} className="mt-1 text-xs text-status-error-text">{errorMessage}</p>
        )}
      </div>
    )
  }
)

Input.displayName = 'Input'
