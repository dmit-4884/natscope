import { useState, useEffect, useRef, useCallback, type KeyboardEvent } from 'react'
import { cn } from '@/utils/cn'
import { formatCount } from '@/utils/formatters'
import { CloseIcon } from './icons'

export interface SearchInputProps {
  /** Search value */
  value: string
  /** Change callback */
  onChange: (value: string) => void
  /** Placeholder text */
  placeholder?: string
  /** Debounce delay in ms (0 to disable) */
  debounce?: number
  /** Search suggestions */
  suggestions?: string[]
  /** Recent searches */
  recentSearches?: string[]
  /** Results count to display */
  resultsCount?: number
  /** Show clear button */
  clearable?: boolean
  /** Size variant */
  size?: 'sm' | 'md'
  /** Loading state */
  loading?: boolean
  /** Autofocus on mount */
  autoFocus?: boolean
  /** Additional classes */
  className?: string
}

/**
 * Search input with debounce and suggestions
 *
 * @example
 * <SearchInput
 *   value={search}
 *   onChange={setSearch}
 *   placeholder="Search messages..."
 *   debounce={300}
 * />
 *
 * @example
 * <SearchInput
 *   value={search}
 *   onChange={setSearch}
 *   suggestions={['orders.*', 'users.*']}
 *   resultsCount={42}
 * />
 */
export function SearchInput({
  value,
  onChange,
  placeholder = 'Search...',
  debounce = 300,
  suggestions,
  recentSearches,
  resultsCount,
  clearable = true,
  size = 'md',
  loading = false,
  autoFocus = false,
  className,
}: SearchInputProps) {
  const [localValue, setLocalValue] = useState(value)
  const [showSuggestions, setShowSuggestions] = useState(false)
  const [selectedIndex, setSelectedIndex] = useState(-1)
  const inputRef = useRef<HTMLInputElement>(null)
  const debounceRef = useRef<ReturnType<typeof setTimeout>>()
  const blurTimeoutRef = useRef<ReturnType<typeof setTimeout>>()

  // Cleanup timeouts on unmount
  useEffect(() => {
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current)
      if (blurTimeoutRef.current) clearTimeout(blurTimeoutRef.current)
    }
  }, [])

  // Sync external value
  useEffect(() => {
    setLocalValue(value)
  }, [value])

  // Debounced onChange
  useEffect(() => {
    if (debounce > 0) {
      debounceRef.current = setTimeout(() => {
        onChange(localValue)
      }, debounce)
      return () => {
        if (debounceRef.current) {
          clearTimeout(debounceRef.current)
        }
      }
    } else {
      onChange(localValue)
    }
  }, [localValue, debounce, onChange])

  const allSuggestions = [
    ...(recentSearches || []),
    ...(suggestions || []),
  ].filter((s, i, arr) => arr.indexOf(s) === i) // dedupe

  const filteredSuggestions = allSuggestions.filter(
    (s) => s.toLowerCase().includes(localValue.toLowerCase()) && s !== localValue
  )

  const handleKeyDown = useCallback(
    (e: KeyboardEvent<HTMLInputElement>) => {
      if (!showSuggestions || filteredSuggestions.length === 0) return

      switch (e.key) {
        case 'ArrowDown':
          e.preventDefault()
          setSelectedIndex((prev) =>
            prev < filteredSuggestions.length - 1 ? prev + 1 : 0
          )
          break
        case 'ArrowUp':
          e.preventDefault()
          setSelectedIndex((prev) =>
            prev > 0 ? prev - 1 : filteredSuggestions.length - 1
          )
          break
        case 'Enter':
          if (selectedIndex >= 0) {
            e.preventDefault()
            setLocalValue(filteredSuggestions[selectedIndex])
            setShowSuggestions(false)
            setSelectedIndex(-1)
          }
          break
        case 'Escape':
          setShowSuggestions(false)
          setSelectedIndex(-1)
          break
      }
    },
    [showSuggestions, filteredSuggestions, selectedIndex]
  )

  const handleClear = () => {
    setLocalValue('')
    onChange('')
    inputRef.current?.focus()
  }

  const sizeClasses = {
    sm: 'px-3 py-1.5 text-xs pl-8',
    md: 'px-4 py-2 text-sm pl-10',
  }

  const iconSize = size === 'sm' ? 'w-4 h-4' : 'w-4 h-4'
  const iconLeft = size === 'sm' ? 'left-2.5' : 'left-3'

  return (
    <div className={cn('relative', className)}>
      {/* Search icon */}
      <span
        className={cn(
          'absolute top-1/2 -translate-y-1/2 text-content-muted pointer-events-none',
          iconLeft
        )}
      >
        {loading ? (
          <svg
            className={cn(iconSize, 'animate-spin')}
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
              d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
            />
          </svg>
        ) : (
          <svg className={iconSize} fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
            />
          </svg>
        )}
      </span>

      {/* Input */}
      <input
        ref={inputRef}
        type="text"
        value={localValue}
        onChange={(e) => setLocalValue(e.target.value)}
        onFocus={() => setShowSuggestions(true)}
        onBlur={() => {
          if (blurTimeoutRef.current) clearTimeout(blurTimeoutRef.current)
          blurTimeoutRef.current = setTimeout(() => setShowSuggestions(false), 200)
        }}
        onKeyDown={handleKeyDown}
        placeholder={placeholder}
        autoFocus={autoFocus}
        className={cn(
          'w-full rounded-md border border-border-strong bg-surface-primary text-content-primary',
          'placeholder:text-content-muted',
          'focus:outline-none focus:ring-2 focus:ring-border-focus focus:border-border-focus',
          sizeClasses[size],
          (clearable && localValue) || resultsCount !== undefined ? 'pr-16' : 'pr-3'
        )}
      />

      {/* Right side: clear button and/or results count */}
      <div className="absolute right-2 top-1/2 -translate-y-1/2 flex items-center gap-2">
        {resultsCount !== undefined && localValue && (
          <span className="text-xs text-content-muted tabular-nums">
            {formatCount(resultsCount)}
          </span>
        )}
        {clearable && localValue && (
          <button
            type="button"
            onClick={handleClear}
            className="p-0.5 text-content-muted hover:text-content-secondary transition-colors"
            aria-label="Clear search"
          >
            <CloseIcon className="w-4 h-4" />
          </button>
        )}
      </div>

      {/* Suggestions dropdown */}
      {showSuggestions && filteredSuggestions.length > 0 && (
        <div className="absolute top-full left-0 right-0 mt-1 bg-surface-primary border border-border rounded-md shadow-dropdown z-50 max-h-48 overflow-auto">
          {filteredSuggestions.map((suggestion, index) => (
            <button
              key={suggestion}
              type="button"
              onClick={() => {
                setLocalValue(suggestion)
                setShowSuggestions(false)
              }}
              className={cn(
                'w-full px-3 py-2 text-left text-sm',
                index === selectedIndex
                  ? 'bg-accent-light text-accent-text'
                  : 'hover:bg-surface-secondary text-gray-700'
              )}
            >
              {suggestion}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}

SearchInput.displayName = 'SearchInput'
