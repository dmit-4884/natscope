import { cloneElement, isValidElement, useId, type ReactElement, type ReactNode } from 'react'
import { Badge } from './Badge'

interface ImmutableFieldProps {
  /** Field label */
  label: string
  /** Whether the field is immutable (disabled in edit mode) */
  isImmutable: boolean
  /** Help text shown below the field */
  helpText?: string
  /** The form control to render */
  children: ReactNode
}

/**
 * Wrapper component for form fields that may be immutable.
 * Provides consistent styling and lock indicator for immutable fields.
 */
export function ImmutableField({
  label,
  isImmutable,
  helpText,
  children,
}: ImmutableFieldProps) {
  const fieldId = useId()
  // Wire label to control via htmlFor/id for a11y; cloning id onto the child
  // is a no-op for elements that don't forward unrecognized DOM props.
  const child =
    label && isValidElement(children)
      ? cloneElement(children as ReactElement<{ id?: string }>, {
          id: (children as ReactElement<{ id?: string }>).props.id ?? fieldId,
        })
      : children

  return (
    <div className={isImmutable ? 'opacity-60' : ''}>
      <div className="flex items-center gap-2 mb-1">
        <label htmlFor={label ? fieldId : undefined} className="block text-sm font-medium text-gray-700">
          {label}
        </label>
        {isImmutable && (
          <Badge variant="warning" size="sm">
            <svg className="w-3 h-3 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
            </svg>
            Immutable
          </Badge>
        )}
      </div>
      <div className={isImmutable ? 'pointer-events-none' : ''}>
        {child}
      </div>
      {helpText && (
        <p className="text-xs text-content-tertiary mt-1">{helpText}</p>
      )}
    </div>
  )
}

ImmutableField.displayName = 'ImmutableField'
