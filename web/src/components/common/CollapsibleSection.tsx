import { useId, useState } from 'react'
import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { ChevronDownIcon } from '@/components/ui'

interface CollapsibleSectionProps {
  title: string
  icon?: ReactNode
  defaultOpen?: boolean
  children: ReactNode
  badge?: number | string
  isActive?: boolean
  onClick?: () => void
  onOpen?: () => void
  linkTo?: string
  actions?: ReactNode
}

export default function CollapsibleSection({
  title,
  icon,
  defaultOpen = true,
  children,
  badge,
  isActive = false,
  onClick,
  onOpen,
  linkTo,
  actions,
}: CollapsibleSectionProps) {
  const [isOpen, setIsOpen] = useState(defaultOpen)
  const panelId = useId()

  const toggle = () => {
    const next = !isOpen
    setIsOpen(next)
    if (next && onOpen) {
      onOpen()
    }
  }

  const handleHeaderClick = () => {
    if (onClick) {
      onClick()
    } else if (!linkTo) {
      toggle()
    }
  }

  // Chevron is a sibling button, not nested — nested interactives are invalid
  // HTML and break screen reader focus order.
  const headerClassName = `flex items-center justify-between text-left transition-colors flex-1 px-3 py-2 ${
    isActive
      ? 'bg-accent-light text-accent-text'
      : 'bg-surface-secondary text-gray-700 hover:bg-surface-tertiary'
  }`

  const headerLabelContent = (
    <div className="flex items-center gap-2">
      {icon && <span className="w-4 h-4" aria-hidden="true">{icon}</span>}
      <span className="text-xs font-semibold uppercase tracking-wide">{title}</span>
      {badge !== undefined && (
        <span className={`text-xs px-1.5 py-0.5 rounded-full ${
          isActive ? 'bg-accent-muted text-accent-text' : 'bg-surface-hover text-content-secondary'
        }`}>
          {badge}
        </span>
      )}
    </div>
  )

  const chevron = <ChevronDownIcon className={`w-4 h-4 transition-transform ${isOpen ? '' : '-rotate-90'}`} />

  return (
    <div className="border-b border-border">
      <div
        className={`flex items-stretch w-full ${
          isActive ? 'bg-accent-light' : 'bg-surface-secondary hover:bg-surface-tertiary'
        }`}
      >
        {linkTo ? (
          <Link to={linkTo} className={headerClassName} onClick={handleHeaderClick}>
            {headerLabelContent}
          </Link>
        ) : (
          <button onClick={handleHeaderClick} className={headerClassName} aria-expanded={isOpen} aria-controls={panelId}>
            {headerLabelContent}
          </button>
        )}
        {actions && (
          <div className="flex items-center" onClick={(e) => e.stopPropagation()}>
            {actions}
          </div>
        )}
        <button
          type="button"
          onClick={toggle}
          aria-expanded={isOpen}
          aria-controls={panelId}
          aria-label={`${isOpen ? 'Collapse' : 'Expand'} ${title}`}
          className={`flex items-center justify-center px-2 ${
            isActive ? 'text-accent-text hover:bg-accent-muted' : 'text-gray-700 hover:bg-surface-hover'
          }`}
        >
          {chevron}
        </button>
      </div>
      {isOpen && <div id={panelId}>{children}</div>}
    </div>
  )
}
