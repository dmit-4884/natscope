import { useState, useRef, useEffect } from 'react'
import { CloseIcon, SearchIcon } from '@/components/ui'
import { toDatetimeLocal, minutesAgoLocal } from './jumpToTime'

export interface FilterValues {
  subject: string
  startSequence: number | null
  startDate: string | null
  contentFilter: string
}

interface AdvancedFiltersProps {
  filters: FilterValues
  availableSubjects?: string[]
  onFiltersChange: (filters: FilterValues) => void
  onClose: () => void
}

export default function AdvancedFilters({
  filters,
  availableSubjects = [],
  onFiltersChange,
  onClose,
}: AdvancedFiltersProps) {
  const [localFilters, setLocalFilters] = useState<FilterValues>(filters)
  const [showSubjectSuggestions, setShowSubjectSuggestions] = useState(false)
  const subjectInputRef = useRef<HTMLInputElement>(null)
  const dropdownRef = useRef<HTMLDivElement>(null)

  // Resync the local draft when the parent's filters actually change (e.g. a
  // chip is removed in the toolbar). Keying on the serialized value avoids
  // clobbering in-progress edits when the prop is a fresh object of equal value.
  const filtersKey = JSON.stringify(filters)
  useEffect(() => {
    setLocalFilters(filters)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filtersKey])

  // Filter suggestions based on input
  const filteredSuggestions = localFilters.subject
    ? availableSubjects.filter((s) =>
        s.toLowerCase().includes(localFilters.subject.toLowerCase())
      )
    : availableSubjects

  // Close dropdown when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(event.target as Node) &&
        !subjectInputRef.current?.contains(event.target as Node)
      ) {
        setShowSubjectSuggestions(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  const handleApply = () => {
    onFiltersChange(localFilters)
    onClose()
  }

  const handleReset = () => {
    const emptyFilters: FilterValues = {
      subject: '',
      startSequence: null,
      startDate: null,
      contentFilter: '',
    }
    setLocalFilters(emptyFilters)
    onFiltersChange(emptyFilters)
  }

  return (
    <div className="absolute top-full left-0 mt-1 w-80 bg-surface-primary border border-border rounded-lg shadow-lg z-30">
      {/* Header */}
      <div className="px-4 py-3 border-b bg-surface-secondary flex items-center justify-between">
        <div className="flex items-center gap-2">
          <SearchIcon className="w-4 h-4 text-content-tertiary" />
          <span className="text-sm font-medium text-gray-700">Filters</span>
        </div>
        <button
          onClick={onClose}
          aria-label="Close filters"
          className="text-content-muted hover:text-content-secondary"
        >
          <CloseIcon className="w-4 h-4" />
        </button>
      </div>

      {/* Filters */}
      <div className="p-4 space-y-4">
        {/* Subject Filter */}
        <div>
          <label className="flex items-center gap-2 text-sm font-medium text-gray-700 mb-1.5">
            <svg className="w-4 h-4 text-content-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z" />
            </svg>
            Subject
          </label>
          <div className="relative">
            <input
              ref={subjectInputRef}
              type="text"
              value={localFilters.subject}
              onChange={(e) => setLocalFilters({ ...localFilters, subject: e.target.value })}
              onFocus={() => setShowSubjectSuggestions(true)}
              placeholder="e.g., orders.* or analytics.>"
              className="w-full px-3 py-2 text-sm border border-border-strong rounded-md focus:ring-2 focus:ring-border-focus focus:border-border-focus"
            />
            {/* Suggestions dropdown */}
            {showSubjectSuggestions && filteredSuggestions.length > 0 && (
              <div
                ref={dropdownRef}
                className="absolute z-40 mt-1 w-full bg-surface-primary border border-border-strong rounded-md shadow-lg max-h-48 overflow-auto"
              >
                {filteredSuggestions.map((subject, idx) => (
                  <button
                    key={idx}
                    onClick={() => {
                      setLocalFilters({ ...localFilters, subject })
                      setShowSubjectSuggestions(false)
                    }}
                    className="w-full px-3 py-2 text-left text-sm hover:bg-accent-light focus:bg-accent-light focus:outline-none truncate font-mono"
                    title={subject}
                  >
                    {localFilters.subject ? (
                      <span>
                        {subject
                          .split(new RegExp(`(${localFilters.subject.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')})`, 'gi'))
                          .map((part, i) =>
                            part.toLowerCase() === localFilters.subject.toLowerCase() ? (
                              <mark key={i} className="bg-yellow-200">
                                {part}
                              </mark>
                            ) : (
                              part
                            )
                          )}
                      </span>
                    ) : (
                      subject
                    )}
                  </button>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* Content Filter (payload search) */}
        <div>
          <label className="flex items-center gap-2 text-sm font-medium text-gray-700 mb-1.5">
            <SearchIcon className="w-4 h-4 text-content-muted" />
            Payload Search
          </label>
          <input
            type="text"
            value={localFilters.contentFilter}
            onChange={(e) => setLocalFilters({ ...localFilters, contentFilter: e.target.value })}
            placeholder="Search in message content..."
            className="w-full px-3 py-2 text-sm border border-border-strong rounded-md focus:ring-2 focus:ring-border-focus focus:border-border-focus"
          />
          <p className="mt-1 text-xs text-content-muted">Case-insensitive search in decoded or raw payload</p>
        </div>

        {/* Start Sequence Filter */}
        <div>
          <label className="flex items-center gap-2 text-sm font-medium text-gray-700 mb-1.5">
            <svg className="w-4 h-4 text-content-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 20l4-16m2 16l4-16M6 9h14M4 15h14" />
            </svg>
            Start Sequence
          </label>
          <input
            type="number"
            value={localFilters.startSequence ?? ''}
            onChange={(e) =>
              setLocalFilters({
                ...localFilters,
                startSequence: e.target.value ? parseInt(e.target.value, 10) : null,
              })
            }
            placeholder="e.g., 1000"
            className="w-full px-3 py-2 text-sm border border-border-strong rounded-md focus:ring-2 focus:ring-border-focus focus:border-border-focus"
          />
        </div>

        {/* Start Date Filter */}
        <div>
          <label className="flex items-center gap-2 text-sm font-medium text-gray-700 mb-1.5">
            <svg className="w-4 h-4 text-content-muted" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
            </svg>
            Start Date & Time
          </label>
          <div className="flex gap-2">
            <div className="flex-1">
              <input
                type="date"
                value={localFilters.startDate?.split('T')[0] ?? ''}
                onChange={(e) => {
                  const time = localFilters.startDate?.split('T')[1] || '00:00'
                  setLocalFilters({
                    ...localFilters,
                    startDate: e.target.value ? `${e.target.value}T${time}` : null,
                  })
                }}
                className="w-full px-3 py-2 text-sm border border-border-strong rounded-md focus:ring-2 focus:ring-border-focus focus:border-border-focus"
              />
            </div>
            <div className="w-24">
              <input
                type="time"
                value={localFilters.startDate?.split('T')[1] ?? ''}
                onChange={(e) => {
                  const date = localFilters.startDate?.split('T')[0] || new Date().toISOString().split('T')[0]
                  setLocalFilters({
                    ...localFilters,
                    startDate: e.target.value ? `${date}T${e.target.value}` : localFilters.startDate,
                  })
                }}
                className="w-full px-2 py-2 text-sm border border-border-strong rounded-md focus:ring-2 focus:ring-border-focus focus:border-border-focus"
              />
            </div>
          </div>
          {/* Quick select buttons */}
          <div className="flex gap-1 mt-2">
            <button
              type="button"
              onClick={() => setLocalFilters({ ...localFilters, startDate: toDatetimeLocal(new Date()) })}
              data-testid="jump-now"
              className="px-2 py-1 text-xs bg-accent-muted text-accent-text hover:bg-blue-200 rounded font-medium"
            >
              Now
            </button>
            <button
              type="button"
              onClick={() => setLocalFilters({ ...localFilters, startDate: minutesAgoLocal(60) })}
              className="px-2 py-1 text-xs bg-surface-tertiary hover:bg-surface-hover rounded"
            >
              1h ago
            </button>
            <button
              type="button"
              onClick={() => setLocalFilters({ ...localFilters, startDate: minutesAgoLocal(24 * 60) })}
              className="px-2 py-1 text-xs bg-surface-tertiary hover:bg-surface-hover rounded"
            >
              24h ago
            </button>
            <button
              type="button"
              onClick={() => setLocalFilters({ ...localFilters, startDate: minutesAgoLocal(7 * 24 * 60) })}
              className="px-2 py-1 text-xs bg-surface-tertiary hover:bg-surface-hover rounded"
            >
              7d ago
            </button>
            {localFilters.startDate && (
              <button
                type="button"
                onClick={() => setLocalFilters({ ...localFilters, startDate: null })}
                className="px-2 py-1 text-xs text-status-error-text hover:bg-status-error-bg rounded ml-auto"
              >
                Clear
              </button>
            )}
          </div>
        </div>
      </div>

      {/* Actions */}
      <div className="px-4 py-3 border-t bg-surface-secondary flex items-center justify-between">
        <button
          onClick={handleReset}
          className="text-sm text-content-secondary hover:text-gray-800"
        >
          Reset
        </button>
        <div className="flex gap-2">
          <button
            onClick={onClose}
            className="px-3 py-1.5 text-sm text-content-secondary hover:bg-surface-tertiary rounded-md"
          >
            Cancel
          </button>
          <button
            onClick={handleApply}
            className="px-3 py-1.5 text-sm bg-accent text-content-inverse hover:bg-accent-hover rounded-md"
          >
            Apply
          </button>
        </div>
      </div>
    </div>
  )
}
