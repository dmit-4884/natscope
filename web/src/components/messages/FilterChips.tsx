import { formatTimestamp } from '@/utils/formatters'
import { FilterChip, FilterChipsGroup } from '@/components/ui'
import type { FilterValues } from './AdvancedFilters'

interface FilterChipsProps {
  filters: FilterValues
  onRemoveFilter: (key: keyof FilterValues) => void
  onClearAll: () => void
}

// startDate carries a date+time value (see AdvancedFilters), so it needs
// the timestamp-with-time formatter, not the date-only one.
function formatFilterDate(dateStr: string): string {
  try {
    return formatTimestamp(dateStr, 'absolute')
  } catch {
    return dateStr
  }
}

export default function FilterChips({ filters, onRemoveFilter, onClearAll }: FilterChipsProps) {
  const activeFilters: Array<{ key: keyof FilterValues; label: string; value: string }> = []

  if (filters.subject) {
    activeFilters.push({
      key: 'subject',
      label: 'Subject',
      value: filters.subject,
    })
  }

  if (filters.startSequence !== null) {
    activeFilters.push({
      key: 'startSequence',
      label: 'From Seq',
      value: filters.startSequence.toString(),
    })
  }

  if (filters.startDate) {
    activeFilters.push({
      key: 'startDate',
      label: 'From',
      value: formatFilterDate(filters.startDate),
    })
  }

  if (filters.contentFilter) {
    activeFilters.push({
      key: 'contentFilter',
      label: 'Content',
      value: filters.contentFilter,
    })
  }

  if (activeFilters.length === 0) {
    return null
  }

  return (
    <FilterChipsGroup onClearAll={activeFilters.length > 1 ? onClearAll : undefined}>
      {activeFilters.map((filter) => (
        <FilterChip
          key={filter.key}
          variant="primary"
          label={filter.label}
          value={filter.value}
          onRemove={() => onRemoveFilter(filter.key)}
        />
      ))}
    </FilterChipsGroup>
  )
}
