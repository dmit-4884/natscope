import { useEffect, useRef, useState, type ReactNode } from 'react'
import Tooltip from '@/components/common/Tooltip'
import { DotsHorizontalIcon } from './icons'

export interface OverflowMenuItem {
  label: string
  onSelect: () => void
  destructive?: boolean
  icon?: ReactNode
  disabled?: boolean
}

interface OverflowMenuProps {
  items: OverflowMenuItem[]
  label?: string
  className?: string
}

export function OverflowMenu({ items, label = 'More actions', className = '' }: OverflowMenuProps) {
  const [open, setOpen] = useState(false)
  const containerRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    const onMouseDown = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) setOpen(false)
    }
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('mousedown', onMouseDown)
    document.addEventListener('keydown', onKeyDown)
    return () => {
      document.removeEventListener('mousedown', onMouseDown)
      document.removeEventListener('keydown', onKeyDown)
    }
  }, [open])

  const normal = items.filter((i) => !i.destructive)
  const destructive = items.filter((i) => i.destructive)

  const renderItem = (item: OverflowMenuItem) => (
    <button
      key={item.label}
      type="button"
      role="menuitem"
      disabled={item.disabled}
      onClick={() => {
        setOpen(false)
        item.onSelect()
      }}
      className={`w-full px-3 py-2 text-sm text-left flex items-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed ${
        item.destructive
          ? 'text-status-error-text hover:bg-status-error-bg'
          : 'text-content-primary hover:bg-surface-secondary'
      }`}
    >
      {item.icon && <span className="w-4 h-4 shrink-0" aria-hidden="true">{item.icon}</span>}
      {item.label}
    </button>
  )

  return (
    <div ref={containerRef} className={`relative ${className}`}>
      <Tooltip content={label}>
        <button
          type="button"
          onClick={() => setOpen((v) => !v)}
          aria-label={label}
          aria-haspopup="menu"
          aria-expanded={open}
          className="p-2 rounded-md text-content-tertiary hover:text-content-primary hover:bg-surface-tertiary transition-colors"
        >
          <DotsHorizontalIcon className="w-4 h-4" />
        </button>
      </Tooltip>
      {open && (
        <div
          role="menu"
          className="absolute right-0 top-full mt-1 w-48 bg-surface-primary border border-border rounded-lg shadow-dropdown z-dropdown overflow-hidden"
        >
          {normal.map(renderItem)}
          {normal.length > 0 && destructive.length > 0 && <div className="border-t border-border" />}
          {destructive.map(renderItem)}
        </div>
      )}
    </div>
  )
}
