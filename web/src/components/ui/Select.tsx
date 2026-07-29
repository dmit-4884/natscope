import { forwardRef, type SelectHTMLAttributes } from 'react'
import { cn } from '@/utils/cn'
import { ChevronDownIcon } from './icons'

interface SelectOption {
  value: string
  label: string
  disabled?: boolean
}

export interface SelectProps extends Omit<SelectHTMLAttributes<HTMLSelectElement>, 'size'> {
  /** Select options */
  options: SelectOption[]
  /** Placeholder text */
  placeholder?: string
  /** Error state */
  error?: boolean
  /** Size variant */
  size?: 'sm' | 'md' | 'lg'
}

const sizeStyles = {
  sm: 'px-2 py-1 text-xs pr-7',
  md: 'px-3 py-2 text-sm pr-8',
  lg: 'px-4 py-3 text-base pr-10',
}

/**
 * Select dropdown component
 *
 * @example
 * <Select
 *   value={selected}
 *   onChange={(e) => setSelected(e.target.value)}
 *   options={[
 *     { value: '25', label: '25' },
 *     { value: '50', label: '50' },
 *     { value: '100', label: '100' },
 *   ]}
 * />
 */
export const Select = forwardRef<HTMLSelectElement, SelectProps>(
  (
    {
      options,
      placeholder,
      error = false,
      size = 'md',
      disabled,
      className,
      ...props
    },
    ref
  ) => {
    return (
      <div className="relative">
        <select
          ref={ref}
          disabled={disabled}
          className={cn(
            'w-full appearance-none rounded-md border bg-surface-primary text-content-primary',
            'transition-colors duration-150',
            'focus:outline-none focus:ring-2 focus:ring-border-focus focus:border-border-focus',
            'disabled:bg-surface-tertiary disabled:text-content-tertiary disabled:cursor-not-allowed',
            error
              ? 'border-status-error-border focus:ring-status-error-border focus:border-status-error-border'
              : 'border-border-strong',
            sizeStyles[size],
            className
          )}
          {...props}
        >
          {placeholder && (
            <option value="" disabled>
              {placeholder}
            </option>
          )}
          {options.map((option) => (
            <option key={option.value} value={option.value} disabled={option.disabled}>
              {option.label}
            </option>
          ))}
        </select>

        {/* Chevron icon */}
        <span className="absolute inset-y-0 right-0 flex items-center pr-2 pointer-events-none">
          <ChevronDownIcon className="w-4 h-4 text-content-muted" />
        </span>
      </div>
    )
  }
)

Select.displayName = 'Select'
