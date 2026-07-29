import { useState, useRef, useEffect, useId, useMemo } from 'react'
import { cn } from '@/utils/cn'
import { ChevronDownIcon, CheckIcon } from './icons'

export interface SearchableSelectOption {
  value: string
  label: string
  disabled?: boolean
}

export interface SearchableSelectProps {
  /** Dropdown options */
  options: SearchableSelectOption[]
  /** Selected value */
  value?: string
  /** Placeholder text */
  placeholder?: string
  /** Search placeholder */
  searchPlaceholder?: string
  /** Disabled state */
  disabled?: boolean
  /** Loading state */
  loading?: boolean
  /** Error state */
  error?: boolean
  /** Change handler */
  onChange?: (value: string) => void
  label?: string
  /** Additional class name */
  className?: string
}

export function SearchableSelect({
  options,
  value,
  placeholder = 'Select...',
  searchPlaceholder = 'Search...',
  disabled = false,
  loading = false,
  error = false,
  onChange,
  label,
  className,
}: SearchableSelectProps) {
  const [isOpen, setIsOpen] = useState(false)
  const [search, setSearch] = useState('')
  const [highlightedIndex, setHighlightedIndex] = useState(-1)
  const containerRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)
  const listRef = useRef<HTMLUListElement>(null)

  const uid = useId()
  const listboxId = `${uid}-listbox`
  const optionId = (index: number) => `${uid}-option-${index}`

  const selectedOption = options.find((opt) => opt.value === value)

  // Filter options by search
  const filteredOptions = useMemo(() => {
    if (!search.trim()) return options
    const searchLower = search.toLowerCase()
    return options.filter((opt) => opt.label.toLowerCase().includes(searchLower))
  }, [options, search])

  // Close on outside click
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
        setIsOpen(false)
        setSearch('')
      }
    }

    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  // Focus input when opened
  useEffect(() => {
    if (isOpen && inputRef.current) {
      inputRef.current.focus()
    }
  }, [isOpen])

  // Reset highlighted index when filtered options change
  useEffect(() => {
    setHighlightedIndex(filteredOptions.length > 0 ? 0 : -1)
  }, [filteredOptions.length])

  // Keyboard navigation
  const handleKeyDown = (event: React.KeyboardEvent) => {
    switch (event.key) {
      case 'ArrowDown':
        event.preventDefault()
        setHighlightedIndex((prev) => {
          const nextIndex = prev + 1
          return nextIndex >= filteredOptions.length ? 0 : nextIndex
        })
        break
      case 'ArrowUp':
        event.preventDefault()
        setHighlightedIndex((prev) => {
          const nextIndex = prev - 1
          return nextIndex < 0 ? filteredOptions.length - 1 : nextIndex
        })
        break
      case 'Enter':
        event.preventDefault()
        if (highlightedIndex >= 0 && !filteredOptions[highlightedIndex]?.disabled) {
          handleSelect(filteredOptions[highlightedIndex].value)
        }
        break
      case 'Escape':
        event.preventDefault()
        setIsOpen(false)
        setSearch('')
        break
    }
  }

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
      if (isOpen) {
        setSearch('')
      }
    }
  }

  const handleSelect = (optionValue: string) => {
    onChange?.(optionValue)
    setIsOpen(false)
    setSearch('')
  }

  return (
    <div ref={containerRef} className={cn('relative', className)}>
      {/* Trigger button */}
      <button
        type="button"
        onClick={handleToggle}
        disabled={disabled || loading}
        aria-haspopup="listbox"
        aria-expanded={isOpen}
        aria-controls={isOpen ? listboxId : undefined}
        aria-label={label}
        className={cn(
          'w-full flex items-center justify-between px-3 py-2 text-sm rounded-lg border bg-surface-primary text-left',
          'transition-colors duration-150',
          'focus:outline-none focus:ring-2 focus:ring-border-focus focus:border-border-focus',
          'disabled:bg-surface-tertiary disabled:text-content-tertiary disabled:cursor-not-allowed',
          error
            ? 'border-status-error-border focus:ring-status-error-border focus:border-status-error-border'
            : 'border-border-strong hover:border-gray-400',
          isOpen && 'ring-2 ring-border-focus border-border-focus'
        )}
      >
        <span className={cn('truncate', !selectedOption && 'text-content-tertiary')}>
          {loading ? 'Loading...' : selectedOption?.label || placeholder}
        </span>
        <ChevronDownIcon
          className={cn('w-4 h-4 ml-2 text-content-muted transition-transform flex-shrink-0', isOpen && 'rotate-180')}
        />
      </button>

      {/* Dropdown menu */}
      {isOpen && (
        <div className="absolute z-50 w-full mt-1 bg-surface-primary border border-border rounded-lg shadow-lg overflow-hidden">
          {/* Search input */}
          <div className="p-2 border-b border-gray-100">
            <div className="relative">
              <svg
                className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-content-muted"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
                aria-hidden="true"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
                />
              </svg>
              <input
                ref={inputRef}
                type="text"
                role="combobox"
                aria-expanded={isOpen}
                aria-controls={listboxId}
                aria-autocomplete="list"
                aria-activedescendant={highlightedIndex >= 0 ? optionId(highlightedIndex) : undefined}
                aria-label={searchPlaceholder}
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                onKeyDown={handleKeyDown}
                placeholder={searchPlaceholder}
                className="w-full pl-9 pr-3 py-2 text-sm border border-border rounded-md focus:outline-none focus:ring-1 focus:ring-border-focus focus:border-border-focus"
              />
            </div>
          </div>

          {/* Options list */}
          <ul
            ref={listRef}
            id={listboxId}
            role="listbox"
            aria-label={label ?? placeholder}
            className="max-h-60 overflow-auto py-1"
          >
            {filteredOptions.length === 0 ? (
              <li role="presentation" className="px-3 py-2 text-sm text-content-tertiary text-center">
                No results found
              </li>
            ) : (
              filteredOptions.map((option, index) => (
                <li
                  key={option.value}
                  id={optionId(index)}
                  role="option"
                  aria-selected={option.value === value}
                  aria-disabled={option.disabled}
                  onClick={() => !option.disabled && handleSelect(option.value)}
                  onMouseEnter={() => setHighlightedIndex(index)}
                  className={cn(
                    'px-3 py-2 text-sm cursor-pointer transition-colors flex items-center gap-2',
                    option.disabled && 'text-content-muted cursor-not-allowed',
                    !option.disabled && highlightedIndex === index && 'bg-accent-light',
                    !option.disabled && option.value === value && 'bg-accent-muted text-blue-900',
                  )}
                >
                  {option.value === value && (
                    <CheckIcon className="w-4 h-4 text-accent flex-shrink-0" />
                  )}
                  <span className={cn(option.value !== value && 'pl-6')}>{option.label}</span>
                </li>
              ))
            )}
          </ul>
        </div>
      )}
    </div>
  )
}

SearchableSelect.displayName = 'SearchableSelect'
