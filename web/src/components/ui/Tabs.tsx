import { useId, useRef, type KeyboardEvent, type ReactNode } from 'react'
import { cn } from '@/utils/cn'

interface Tab {
  value: string
  label: string
  icon?: ReactNode
  trailing?: ReactNode
  count?: number
  disabled?: boolean
}

export interface TabsProps {
  /** Currently selected tab value */
  value: string
  /** Callback when tab changes */
  onChange: (value: string) => void
  /** Tab definitions */
  tabs: Tab[]
  /** Tab variant style */
  variant?: 'default' | 'underline' | 'pills'
  label?: string
  idPrefix?: string
  /** Additional classes for container */
  className?: string
}

export function Tabs({
  value,
  onChange,
  tabs,
  variant = 'default',
  label,
  idPrefix,
  className,
}: TabsProps) {
  const fallbackId = useId()
  const prefix = idPrefix ?? fallbackId
  const listRef = useRef<HTMLDivElement>(null)

  const focusTab = (nextValue: string) => {
    onChange(nextValue)
    listRef.current?.querySelector<HTMLButtonElement>(`#${CSS.escape(`${prefix}-tab-${nextValue}`)}`)?.focus()
  }

  const handleKeyDown = (e: KeyboardEvent<HTMLDivElement>) => {
    const enabled = tabs.filter((t) => !t.disabled)
    if (enabled.length === 0) return

    const current = enabled.findIndex((t) => t.value === value)
    let next = -1

    switch (e.key) {
      case 'ArrowRight':
      case 'ArrowDown':
        next = (current + 1) % enabled.length
        break
      case 'ArrowLeft':
      case 'ArrowUp':
        next = (current - 1 + enabled.length) % enabled.length
        break
      case 'Home':
        next = 0
        break
      case 'End':
        next = enabled.length - 1
        break
      default:
        return
    }

    e.preventDefault()
    focusTab(enabled[next].value)
  }

  return (
    <div
      ref={listRef}
      className={cn(
        'flex',
        variant === 'underline' && 'border-b border-border gap-1',
        variant === 'default' && 'gap-1',
        variant === 'pills' && 'gap-2 p-1 bg-surface-tertiary rounded-lg',
        className
      )}
      role="tablist"
      aria-label={label}
      aria-orientation="horizontal"
      onKeyDown={handleKeyDown}
    >
      {tabs.map((tab) => (
        <button
          key={tab.value}
          type="button"
          role="tab"
          id={`${prefix}-tab-${tab.value}`}
          aria-selected={value === tab.value}
          aria-controls={idPrefix ? `${prefix}-panel-${tab.value}` : undefined}
          tabIndex={value === tab.value ? 0 : -1}
          disabled={tab.disabled}
          onClick={() => onChange(tab.value)}
          className={cn(
            'inline-flex items-center gap-2 font-medium transition-colors',
            'focus:outline-none focus-visible:ring-2 focus-visible:ring-border-focus focus-visible:ring-offset-1',
            'disabled:opacity-50 disabled:cursor-not-allowed',
            // Default variant
            variant === 'default' && [
              'px-4 py-2 text-sm rounded-t',
              value === tab.value
                ? 'bg-surface-primary text-accent border-t border-x border-border'
                : 'text-content-secondary hover:text-content-primary hover:bg-surface-tertiary',
            ],
            // Underline variant
            variant === 'underline' && [
              'px-4 py-2 text-sm relative',
              value === tab.value
                ? 'text-accent'
                : 'text-content-tertiary hover:text-gray-700',
            ],
            // Pills variant
            variant === 'pills' && [
              'px-3 py-1.5 text-sm rounded-md',
              value === tab.value
                ? 'bg-surface-primary text-content-primary shadow-sm'
                : 'text-content-secondary hover:text-content-primary',
            ]
          )}
        >
          {tab.icon && <span className="w-4 h-4">{tab.icon}</span>}
          {tab.label}
          {tab.trailing}
          {tab.count !== undefined && (
            <span
              className={cn(
                'px-1.5 py-0.5 text-xs rounded-full',
                value === tab.value
                  ? 'bg-accent-muted text-accent'
                  : 'bg-surface-hover text-content-secondary'
              )}
            >
              {tab.count}
            </span>
          )}
          {/* Underline indicator */}
          {variant === 'underline' && value === tab.value && (
            <span className="absolute bottom-0 left-0 right-0 h-0.5 bg-accent" />
          )}
        </button>
      ))}
    </div>
  )
}

Tabs.displayName = 'Tabs'
