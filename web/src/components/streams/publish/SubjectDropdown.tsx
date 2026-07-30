import { useState, useEffect, useId, useLayoutEffect, useRef } from 'react'
import { createPortal } from 'react-dom'
import { ChevronDownIcon } from '@/components/ui'

interface Props {
  value: string
  options: string[]
  placeholder: string
  onChange: (val: string) => void
}

/**
 * Portal-based subject-pattern dropdown that renders options above the stacking
 * context.
 */
export function SubjectDropdown({ value, options, placeholder, onChange }: Props) {
  const [open, setOpen] = useState(false)
  const [custom, setCustom] = useState('')
  const btnRef = useRef<HTMLButtonElement>(null)
  const listRef = useRef<HTMLDivElement>(null)
  const [pos, setPos] = useState({ top: 0, left: 0, width: 0 })
  const listboxId = `${useId()}-subjects`

  const close = () => {
    setOpen(false)
    btnRef.current?.focus()
  }

  const commitCustom = () => {
    const subject = custom.trim()
    if (!subject) return
    onChange(subject)
    setCustom('')
    close()
  }

  useLayoutEffect(() => {
    if (open && btnRef.current) {
      const rect = btnRef.current.getBoundingClientRect()
      setPos({ top: rect.bottom + 4, left: rect.left, width: rect.width })
    }
  }, [open])

  useEffect(() => {
    if (!open) return
    const handler = (e: MouseEvent) => {
      const target = e.target as Node
      if (btnRef.current?.contains(target) || listRef.current?.contains(target)) return
      setOpen(false)
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [open])

  return (
    <>
      <button
        ref={btnRef}
        type="button"
        onClick={() => setOpen(!open)}
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-controls={open ? listboxId : undefined}
        className="relative w-full px-3 py-2 pr-8 border border-border-strong rounded-lg bg-surface-primary text-sm text-left focus:ring-2 focus:ring-border-focus focus:border-border-focus focus:outline-none"
      >
        <span className={value ? 'text-content-primary' : 'text-content-muted'}>{value || placeholder}</span>
        <ChevronDownIcon className="absolute right-2.5 top-1/2 -translate-y-1/2 w-4 h-4 text-content-muted pointer-events-none" />
      </button>
      {open &&
        createPortal(
          <div
            ref={listRef}
            onKeyDown={(e) => {
              if (e.key === 'Escape') {
                e.preventDefault()
                close()
              }
            }}
            className="fixed bg-surface-primary border border-border rounded-lg shadow-lg"
            style={{ top: pos.top, left: pos.left, width: pos.width, zIndex: 9999 }}
          >
            <div role="listbox" id={listboxId} aria-label={placeholder} className="max-h-60 overflow-auto">
              {value && (
                <button
                  type="button"
                  role="option"
                  aria-selected={false}
                  onClick={() => {
                    onChange('')
                    close()
                  }}
                  className="w-full px-3 py-2 text-sm text-left text-content-muted hover:bg-surface-secondary"
                >
                  {placeholder}
                </button>
              )}
              {options.map((opt) => (
                <button
                  key={opt}
                  type="button"
                  role="option"
                  aria-selected={opt === value}
                  onClick={() => {
                    onChange(opt)
                    close()
                  }}
                  className={`w-full px-3 py-2 text-sm text-left transition-colors ${
                    opt === value ? 'bg-accent-light text-accent-text font-medium' : 'text-content-primary hover:bg-surface-secondary'
                  }`}
                >
                  {opt}
                </button>
              ))}
              {options.length === 0 && (
                <div role="presentation" className="px-3 py-2 text-sm text-content-muted">No subjects available</div>
              )}
            </div>
            <div className="border-t border-border p-2">
              <input
                type="text"
                value={custom}
                onChange={(e) => setCustom(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    e.preventDefault()
                    commitCustom()
                  }
                }}
                placeholder="Custom subject… (Enter to use)"
                aria-label="Custom subject"
                className="w-full px-2 py-1.5 text-sm font-mono border border-border-strong rounded focus:ring-1 focus:ring-border-focus focus:border-border-focus focus:outline-none"
              />
            </div>
          </div>,
          document.body,
        )}
    </>
  )
}
