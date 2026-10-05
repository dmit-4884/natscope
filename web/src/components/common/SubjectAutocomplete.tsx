import { useEffect, useId, useMemo, useRef, useState } from 'react'

interface Props {
  value: string
  onChange: (val: string) => void
  options: string[]
  placeholder?: string
  className?: string
  /** Limit the number of suggestions rendered. */
  maxOptions?: number
  inputId?: string
  disabled?: boolean
  invalid?: boolean
  describedBy?: string
}

/**
 * Free-text subject input with substring-match autocomplete (prefix matches
 * sorted first).
 */
export function SubjectAutocomplete({
  value,
  onChange,
  options,
  placeholder,
  className,
  maxOptions = 8,
  inputId,
  disabled,
  invalid,
  describedBy,
}: Props) {
  const [open, setOpen] = useState(false)
  const [highlight, setHighlight] = useState(0)
  const wrapRef = useRef<HTMLDivElement>(null)
  const listId = useId()

  // Close on outside click.
  useEffect(() => {
    if (!open) return
    const handler = (e: MouseEvent) => {
      if (wrapRef.current && !wrapRef.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [open])

  const filtered = useMemo(() => {
    const q = value.trim().toLowerCase()
    if (!q) return options.slice(0, maxOptions)
    const prefix: string[] = []
    const substr: string[] = []
    for (const o of options) {
      const lo = o.toLowerCase()
      if (lo === q) continue // exact match — no point suggesting "this"
      if (lo.startsWith(q)) prefix.push(o)
      else if (lo.includes(q)) substr.push(o)
    }
    return [...prefix, ...substr].slice(0, maxOptions)
  }, [value, options, maxOptions])

  const firstHighlight = value.trim() ? 0 : -1

  // Reset highlight when options change.
  useEffect(() => {
    setHighlight(firstHighlight)
  }, [filtered, firstHighlight])

  const showDropdown = open && filtered.length > 0

  const choose = (opt: string) => {
    onChange(opt)
    setOpen(false)
  }

  const onKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (!open && (e.key === 'ArrowDown' || e.key === 'ArrowUp')) {
      setOpen(true)
      e.preventDefault()
      return
    }
    if (!showDropdown) return
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      setHighlight((h) => (h + 1) % filtered.length)
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      setHighlight((h) => (h <= 0 ? filtered.length - 1 : h - 1))
    } else if (e.key === 'Enter' && !e.metaKey && !e.ctrlKey && highlight >= 0) {
      e.preventDefault()
      choose(filtered[highlight])
    } else if (e.key === 'Escape') {
      setOpen(false)
    }
  }

  return (
    <div ref={wrapRef} className="relative">
      <input
        id={inputId}
        type="text"
        value={value}
        onChange={(e) => {
          onChange(e.target.value)
          setOpen(true)
        }}
        onFocus={() => setOpen(true)}
        onBlur={() => setOpen(false)}
        onKeyDown={onKeyDown}
        role="combobox"
        aria-expanded={showDropdown}
        aria-controls={listId}
        aria-autocomplete="list"
        aria-activedescendant={showDropdown && highlight >= 0 ? `${listId}-${highlight}` : undefined}
        placeholder={placeholder}
        disabled={disabled}
        aria-invalid={invalid || undefined}
        aria-describedby={describedBy}
        autoComplete="off"
        spellCheck={false}
        className={
          className ??
          'w-full rounded-md border border-border-strong px-3 py-2 text-sm font-mono focus:border-border-focus focus:ring-1 focus:ring-border-focus focus:outline-none disabled:bg-surface-secondary disabled:text-content-tertiary'
        }
      />
      {showDropdown && (
        <ul
          id={listId}
          role="listbox"
          className="absolute z-dropdown mt-1 w-full max-h-60 overflow-auto rounded-md border border-border bg-surface-primary shadow-lg"
        >
          {filtered.map((opt, i) => (
            <li
              key={opt}
              id={`${listId}-${i}`}
              role="option"
              aria-selected={i === highlight}
              onMouseDown={(e) => {
                // mousedown so the input doesn't lose focus + dropdown re-close
                // before click.
                e.preventDefault()
                choose(opt)
              }}
              onMouseEnter={() => setHighlight(i)}
              className={`block w-full cursor-pointer text-left px-3 py-1.5 text-xs font-mono ${
                i === highlight ? 'bg-accent-light text-accent-text' : 'text-gray-700 hover:bg-surface-secondary'
              }`}
            >
              {opt}
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
