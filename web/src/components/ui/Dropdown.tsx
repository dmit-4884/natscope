import { useState, useRef, useEffect, useId } from 'react'
import { cn } from '@/utils/cn'
import { ChevronDownIcon } from './icons'

interface DropdownOption {
  value: string
  label: string
  disabled?: boolean
  /** Optional icon to show before label */
  icon?: React.ReactNode
  /** Optional element to show at the right side */
  rightElement?: React.ReactNode
}

export interface DropdownProps {
  /** Dropdown options */
  options: DropdownOption[]
  /** Selected value */
  value?: string
  /** Placeholder text */
  placeholder?: string
  /** Disabled state */
  disabled?: boolean
  /** Loading state */
  loading?: boolean
  /** Error state */
  error?: boolean
  /** Size variant */
  size?: 'sm' | 'md' | 'lg'
  /** Change handler */
  onChange?: (value: string) => void
  /** Select handler (alternative to onChange, gets value string) */
  onSelect?: (value: string) => void
  /** Dropdown alignment */
  align?: 'left' | 'right'
  label?: string
  /** Additional class name */
  className?: string
  /** id applied to the trigger button — lets a <label htmlFor> reference it */
  id?: string
}

const sizeStyles = {
  sm: 'px-2 py-1.5 text-xs',
  md: 'px-3 py-2 text-sm',
  lg: 'px-4 py-3 text-base',
}

const optionSizeStyles = {
  sm: 'px-2 py-1.5 text-xs',
  md: 'px-3 py-2 text-sm',
  lg: 'px-4 py-3 text-base',
}

export function Dropdown({
  options,
  value,
  placeholder = 'Select...',
  disabled = false,
  loading = false,
  error = false,
  size = 'md',
  onChange,
  onSelect,
  align = 'left',
  label,
  className,
  id,
}: DropdownProps) {
  const [isOpen, setIsOpen] = useState(false)
  const [highlightedIndex, setHighlightedIndex] = useState(-1)
  const containerRef = useRef<HTMLDivElement>(null)
  const listRef = useRef<HTMLUListElement>(null)

  const uid = useId()
  const listboxId = `${uid}-listbox`
  const optionId = (index: number) => `${uid}-option-${index}`

  const selectedOption = options.find((opt) => opt.value === value)

  // Close on outside click
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
        setIsOpen(false)
      }
    }

    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  // Keep the highlighted index in range when options change
  useEffect(() => {
    setHighlightedIndex((prev) => (prev >= options.length ? options.length - 1 : prev))
  }, [options.length])

  // Keyboard navigation
  useEffect(() => {
    if (!isOpen) return

    const handleKeyDown = (event: KeyboardEvent) => {
      switch (event.key) {
        case 'ArrowDown':
          event.preventDefault()
          setHighlightedIndex((prev) => {
            const nextIndex = prev + 1
            return nextIndex >= options.length ? 0 : nextIndex
          })
          break
        case 'ArrowUp':
          event.preventDefault()
          setHighlightedIndex((prev) => {
            const nextIndex = prev - 1
            return nextIndex < 0 ? options.length - 1 : nextIndex
          })
          break
        case 'Enter': {
          event.preventDefault()
          const opt = options[highlightedIndex]
          if (opt && !opt.disabled) {
            onChange?.(opt.value)
            onSelect?.(opt.value)
            setIsOpen(false)
          }
          break
        }
        case 'Escape':
          event.preventDefault()
          setIsOpen(false)
          break
      }
    }

    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [isOpen, highlightedIndex, options, onChange, onSelect])

  // Scroll highlighted option into view
  useEffect(() => {
    if (isOpen && highlightedIndex >= 0 && listRef.current) {
      const highlightedElement = listRef.current.children[highlightedIndex] as HTMLElement
      highlightedElement?.scrollIntoView({ block: 'nearest' })
    }
  }, [highlightedIndex, isOpen])

  const handleToggle = () => {
    if (!disabled && !loading) {
      setIsOpen(!isOpen)
      if (!isOpen) {
        const currentIndex = options.findIndex((opt) => opt.value === value)
        setHighlightedIndex(currentIndex >= 0 ? currentIndex : 0)
      }
    }
  }

  const handleSelect = (optionValue: string) => {
    onChange?.(optionValue)
    onSelect?.(optionValue)
    setIsOpen(false)
  }

  return (
    <div
      ref={containerRef}
      className={cn('relative', className)}
      onBlur={(e) => {
        if (!e.currentTarget.contains(e.relatedTarget as Node)) {
          setIsOpen(false)
        }
      }}
    >
      <button
        type="button"
        id={id}
        onClick={handleToggle}
        disabled={disabled || loading}
        aria-haspopup="listbox"
        aria-expanded={isOpen}
        aria-controls={isOpen ? listboxId : undefined}
        aria-activedescendant={isOpen && highlightedIndex >= 0 ? optionId(highlightedIndex) : undefined}
        aria-label={label}
        className={cn(
          'w-full flex items-center justify-between rounded-md border bg-surface-primary text-left',
          'transition-colors duration-150',
          'focus:outline-none focus:ring-2 focus:ring-border-focus focus:border-border-focus',
          'disabled:bg-surface-tertiary disabled:text-content-tertiary disabled:cursor-not-allowed',
          error
            ? 'border-status-error-border focus:ring-status-error-border focus:border-status-error-border'
            : 'border-border-strong hover:border-gray-400',
          sizeStyles[size]
        )}
      >
        <span className={cn('truncate', !selectedOption && 'text-content-tertiary')}>
          {loading ? 'Loading...' : selectedOption?.label || placeholder}
        </span>
        <ChevronDownIcon
          className={cn('w-4 h-4 ml-2 text-content-muted transition-transform flex-shrink-0', isOpen && 'rotate-180')}
        />
      </button>

      {isOpen && options.length > 0 && (
        <ul
          ref={listRef}
          id={listboxId}
          role="listbox"
          aria-label={label ?? placeholder}
          // Options aren't focusable; a plain mousedown would shift focus off
          // the trigger and fire onBlur before the click lands, closing the list.
          onMouseDown={(e) => e.preventDefault()}
          className={cn(
            'absolute z-50 mt-1 w-full bg-surface-primary border border-border rounded-md shadow-lg',
            'max-h-60 overflow-auto',
            'py-1',
            align === 'right' ? 'right-0' : 'left-0'
          )}
        >
          {options.map((option, index) => (
            <li
              key={option.value}
              id={optionId(index)}
              role="option"
              aria-selected={option.value === value}
              aria-disabled={option.disabled || undefined}
              onClick={() => !option.disabled && handleSelect(option.value)}
              onMouseEnter={() => setHighlightedIndex(index)}
              className={cn(
                'cursor-pointer transition-colors flex items-center gap-2',
                optionSizeStyles[size],
                option.disabled && 'text-content-tertiary cursor-not-allowed',
                !option.disabled && highlightedIndex === index && 'bg-accent-light',
                !option.disabled && option.value === value && 'bg-accent-muted text-blue-900 font-medium',
                !option.disabled && option.value !== value && highlightedIndex !== index && 'hover:bg-surface-secondary'
              )}
            >
              {option.icon && <span className="flex-shrink-0">{option.icon}</span>}
              <span className="flex-1 truncate">{option.label}</span>
              {option.rightElement && <span className="flex-shrink-0 ml-2">{option.rightElement}</span>}
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

Dropdown.displayName = 'Dropdown'
