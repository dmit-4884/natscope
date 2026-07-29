import { useState, useCallback, useRef, useEffect } from 'react'
import { cn } from '@/utils/cn'
import Tooltip from '@/components/common/Tooltip'
import { CheckIcon } from './icons'

export interface CopyButtonProps {
  /** Value to copy (string or function that returns string) */
  value: string | (() => string)
  /** Button variant */
  variant?: 'icon' | 'button' | 'inline'
  /** Size */
  size?: 'sm' | 'md'
  /** Success message */
  successMessage?: string
  /** Button label (for button variant) */
  label?: string
  /** Callback after successful copy */
  onCopy?: () => void
  /** Additional classes */
  className?: string
}

const CopyIcon = () => (
  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"
    />
  </svg>
)

/**
 * Copy button component with success feedback
 *
 * @example
 * // Icon-only copy button
 * <CopyButton value={textToCopy} />
 *
 * @example
 * // Button with label
 * <CopyButton value={jsonData} variant="button" label="Copy JSON" />
 *
 * @example
 * // Inline small copy button
 * <CopyButton value={url} variant="inline" size="sm" />
 */
export function CopyButton({
  value,
  variant = 'icon',
  size = 'md',
  successMessage = 'Copied!',
  label = 'Copy',
  onCopy,
  className,
}: CopyButtonProps) {
  const [copied, setCopied] = useState(false)
  const [error, setError] = useState(false)
  const copiedTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const errorTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  // Cleanup timeouts on unmount
  useEffect(() => {
    return () => {
      if (copiedTimeoutRef.current) clearTimeout(copiedTimeoutRef.current)
      if (errorTimeoutRef.current) clearTimeout(errorTimeoutRef.current)
    }
  }, [])

  const handleCopy = useCallback(async () => {
    try {
      const textToCopy = typeof value === 'function' ? value() : value
      await navigator.clipboard.writeText(textToCopy)
      setCopied(true)
      setError(false)
      onCopy?.()
      if (copiedTimeoutRef.current) clearTimeout(copiedTimeoutRef.current)
      copiedTimeoutRef.current = setTimeout(() => setCopied(false), 2000)
    } catch (err) {
      console.error('Failed to copy:', err)
      setError(true)
      if (errorTimeoutRef.current) clearTimeout(errorTimeoutRef.current)
      errorTimeoutRef.current = setTimeout(() => setError(false), 2000)
    }
  }, [value, onCopy])

  const sizeClasses = {
    sm: 'p-1',
    md: 'p-1.5',
  }

  const buttonSizeClasses = {
    sm: 'px-2 py-1 text-xs gap-1',
    md: 'px-3 py-1.5 text-sm gap-1.5',
  }

  if (variant === 'icon') {
    return (
      <Tooltip content={copied ? successMessage : label}>
        <button
          type="button"
          onClick={handleCopy}
          className={cn(
            'rounded transition-colors',
            sizeClasses[size],
            copied
              ? 'text-status-success-text'
              : error
                ? 'text-status-error-text'
                : 'text-content-muted hover:text-content-secondary hover:bg-surface-tertiary',
            className
          )}
          aria-label={copied ? successMessage : label}
        >
          {copied ? <CheckIcon /> : <CopyIcon />}
        </button>
      </Tooltip>
    )
  }

  if (variant === 'inline') {
    return (
      <button
        type="button"
        onClick={handleCopy}
        className={cn(
          'inline-flex items-center gap-1 rounded transition-colors',
          size === 'sm' ? 'p-0.5' : 'p-1',
          copied
            ? 'text-status-success-text'
            : error
              ? 'text-status-error-text'
              : 'text-content-muted hover:text-content-secondary',
          className
        )}
        aria-label={copied ? successMessage : label}
      >
        {copied ? (
          <CheckIcon className="w-3 h-3" />
        ) : (
          <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"
            />
          </svg>
        )}
      </button>
    )
  }

  // Button variant
  return (
    <button
      type="button"
      onClick={handleCopy}
      className={cn(
        'inline-flex items-center rounded-md transition-colors',
        buttonSizeClasses[size],
        copied
          ? 'bg-status-success-light text-green-700'
          : error
            ? 'bg-status-error-light text-red-700'
            : 'bg-surface-tertiary text-gray-700 hover:bg-surface-hover',
        className
      )}
    >
      {copied ? <CheckIcon /> : <CopyIcon />}
      <span>{copied ? successMessage : label}</span>
    </button>
  )
}

CopyButton.displayName = 'CopyButton'
