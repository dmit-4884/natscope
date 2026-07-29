import { useMemo } from 'react'
import { useProtoSources } from '@/contexts/proto'
import { Dropdown } from '@/components/ui'

interface Props {
  value: string
  onChange: (sourceId: string) => void
  /** When true, only enabled sources are shown. Default true. */
  onlyEnabled?: boolean
  className?: string
  placeholder?: string
  id?: string
  disabled?: boolean
  /** Visual size, mapped to Dropdown size. Default `md`. */
  size?: 'sm' | 'md' | 'lg'
}

/**
 * SourcePicker — single-select dropdown of proto sources.
 *
 * Built on top of the project's custom `Dropdown` (not native <select>) so it
 * matches the rest of the design system: rounded-md trigger with chevron, popup
 * list with hover/focus states, keyboard navigation, click-outside.
 */
export function SourcePicker({
  value,
  onChange,
  onlyEnabled = true,
  className,
  placeholder = 'Select source…',
  disabled,
  size = 'md',
}: Props) {
  const { data: sources, isLoading } = useProtoSources()

  const options = useMemo(() => {
    const visible = onlyEnabled ? (sources ?? []).filter((s) => s.enabled) : (sources ?? [])
    return visible.map((src) => ({
      value: src.id,
      label: src.sourceType === 'local' ? `${src.name} (local)` : src.name,
    }))
  }, [sources, onlyEnabled])

  return (
    <Dropdown
      options={options}
      value={value}
      onChange={onChange}
      placeholder={isLoading ? 'Loading sources…' : placeholder}
      disabled={disabled || isLoading || options.length === 0}
      loading={isLoading}
      size={size}
      className={className}
    />
  )
}
